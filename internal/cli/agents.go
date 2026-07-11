package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/agentblob"
	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/link"
	"github.com/noviopenworks/homonto/internal/merge"
	"github.com/noviopenworks/homonto/internal/subagentpath"
	"github.com/spf13/cobra"
)

func agentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Inspect lifecycle-managed agents",
	}
	cmd.AddCommand(agentsListCmd())
	cmd.AddCommand(agentsAddCmd())
	cmd.AddCommand(agentsUpdateCmd())
	cmd.AddCommand(agentsDoctorCmd())
	return cmd
}

// agentsDoctorCmd builds "agents doctor": a strictly read-only drift report
// comparing declared agents (config) against installed agents (the
// .homonto/agents-lock.json lockfile) and their on-disk files. It writes
// nothing. On a healthy workspace it prints "healthy" and returns nil;
// otherwise it prints each finding and returns a summary error so main exits
// non-zero — mirroring "onto doctor".
func agentsDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report declared-vs-installed agent drift (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			cfgDir := filepath.Dir(cfgPath)
			homontoDir := filepath.Join(cfgDir, ".homonto")

			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			lock, err := agentlock.Load(homontoDir)
			if err != nil {
				return err
			}

			var findings []string

			// 1. declared agents, in sorted name order for deterministic output.
			names := make([]string, 0, len(c.Agents))
			for n := range c.Agents {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, name := range names {
				ag := c.Agents[name]
				inst, installed := lock.Agents[name]
				if !installed {
					findings = append(findings, fmt.Sprintf("%s: declared but not installed (run `homonto agents add %s`)", name, name))
					continue
				}

				// source drift (local: only)
				if strings.HasPrefix(ag.Source, "local:") {
					srcPath := filepath.Join(cfgDir, "homonto", "agents", strings.TrimPrefix(ag.Source, "local:")+".md")
					b, rerr := os.ReadFile(srcPath)
					switch {
					case rerr != nil:
						findings = append(findings, fmt.Sprintf("%s: source file %s missing or unreadable", name, srcPath))
					case len(inst.Installed) > 0:
						// Every target records the same content hash at install, so
						// compare against the first recorded target's hash (sorted for
						// determinism).
						if agentlock.HashContent(b) != firstRecordedHash(inst.Installed) {
							findings = append(findings, fmt.Sprintf("%s: source changed since install (re-run `homonto agents add %s`)", name, name))
						}
					}
				}

				// declared targets present + intact
				declared := ag.TargetsOrAll()
				for _, tool := range sortedStrings(declared) {
					ti, ok := inst.Installed[tool]
					if !ok {
						findings = append(findings, fmt.Sprintf("%s: target %s declared but not installed", name, tool))
						continue
					}
					if _, lerr := os.Lstat(ti.Path); lerr != nil {
						findings = append(findings, fmt.Sprintf("%s (%s): installed file missing: %s", name, tool, ti.Path))
						continue
					}
					// In the three-way-merge model a locally-edited install
					// (on-disk content differing from the recorded base) is a
					// normal, mergeable state and is NOT a problem. A leftover
					// <dst>.merged sidecar, however, marks an unresolved conflict.
					if _, err := os.Lstat(ti.Path + ".merged"); err == nil {
						findings = append(findings, fmt.Sprintf("%s (%s): conflicted (resolve %s.merged, then re-run `homonto agents update %s`)", name, tool, ti.Path, name))
					}
					// link mode: presence via Lstat is sufficient this increment.
				}

				// installed targets no longer declared
				for _, tool := range sortedKeys(inst.Installed) {
					if !containsStr(declared, tool) {
						findings = append(findings, fmt.Sprintf("%s: target %s installed but no longer targeted", name, tool))
					}
				}
			}

			// 2. orphans: installed agents no longer declared.
			for _, name := range sortedKeysAgents(lock.Agents) {
				if _, ok := c.Agents[name]; !ok {
					findings = append(findings, fmt.Sprintf("%s: installed but no longer declared (orphan)", name))
				}
			}

			// verdict
			if len(findings) == 0 {
				cmd.Println("healthy")
				return nil
			}
			for _, f := range findings {
				cmd.Println(f)
			}
			return fmt.Errorf("homonto agents doctor: %d problem(s) found", len(findings))
		},
	}
}

// firstRecordedHash returns the hash of the first install by sorted tool key.
// All targets record the same content hash at install, so any one suffices.
func firstRecordedHash(installed map[string]agentlock.Install) string {
	for _, tool := range sortedKeys(installed) {
		return installed[tool].Hash
	}
	return ""
}

func containsStr(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func sortedStrings(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]agentlock.Install) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeysAgents(m map[string]agentlock.Agent) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func agentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List declared agents (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(c.Agents))
			for n := range c.Agents {
				names = append(names, n)
			}
			sort.Strings(names)
			if len(names) == 0 {
				cmd.Println("No agents declared.")
				return nil
			}
			for _, n := range names {
				ag := c.Agents[n]
				v := ag.Version
				if v == "" {
					v = "unpinned"
				}
				cmd.Printf("%s: %s  version=%s  targets=%s  mode=%s\n",
					n, ag.Source, v, strings.Join(ag.TargetsOrAll(), ","), ag.ModeOrDefault())
			}
			return nil
		},
	}
}

// agentsAddCmd installs a declared local: agent into each target tool's user
// agent dir and records it in .homonto/agents-lock.json. It is conflict-safe
// (a foreign file at any target refuses the whole install) and idempotent (an
// up-to-date target is left untouched).
func agentsAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Install a declared local agent (copy or link) and record it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfgPath, _ := cmd.Flags().GetString("config")
			cfgDir := filepath.Dir(cfgPath)
			homontoDir := filepath.Join(cfgDir, ".homonto")

			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			ag, ok := c.Agents[name]
			if !ok {
				return fmt.Errorf("agents add: agent %q is not declared", name)
			}
			if !strings.HasPrefix(ag.Source, "local:") {
				return fmt.Errorf("agents add: only local: sources are supported yet (got %q)", ag.Source)
			}
			srcName := strings.TrimPrefix(ag.Source, "local:")
			srcPath := filepath.Join(cfgDir, "homonto", "agents", srcName+".md")
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("agents add: source file %s: %w", srcPath, err)
			}
			hash := agentlock.HashContent(content)

			lock, err := agentlock.Load(homontoDir)
			if err != nil {
				return err
			}
			home, _ := os.UserHomeDir()
			mode := ag.ModeOrDefault()
			targets := ag.TargetsOrAll()
			prevInstalled := lock.Agents[name].Installed

			// dstFor returns the install destination for a tool.
			dstFor := func(tool string) string {
				return filepath.Join(subagentpath.Dir(tool, "user", home, ""), name+".md")
			}

			// Pass 1 — conflict scan across all targets. A destination is ours iff
			// the lockfile records this agent at exactly that path; anything else
			// present is a foreign file. Any conflict refuses before writing.
			var conflicts []string
			for _, tool := range targets {
				dst := dstFor(tool)
				prev, recorded := prevInstalled[tool]
				wasManaged := recorded && prev.Path == dst
				if _, err := os.Lstat(dst); err == nil && !wasManaged {
					conflicts = append(conflicts, dst)
				}
			}
			if len(conflicts) > 0 {
				return fmt.Errorf("agents add: %q would clobber unmanaged file(s): %s; installing nothing",
					name, strings.Join(conflicts, ", "))
			}

			// Pass 2 — install + record.
			installed := map[string]agentlock.Install{}
			for _, tool := range targets {
				dst := dstFor(tool)
				prev, recorded := prevInstalled[tool]
				var status string
				switch mode {
				case "copy":
					if _, err := os.Lstat(dst); err == nil && recorded && prev.Hash == hash {
						status = "up to date"
					} else {
						if err := fsutil.WriteAtomic(dst, content); err != nil {
							return err
						}
						if recorded {
							status = "updated"
						} else {
							status = "installed"
						}
					}
				default: // link
					if isSymlinkTo(dst, srcPath) {
						status = "up to date"
					} else {
						if _, err := link.Link(srcPath, dst, homontoDir); err != nil {
							return err
						}
						if recorded {
							status = "updated"
						} else {
							status = "installed"
						}
					}
				}
				installed[tool] = agentlock.Install{Path: dst, Hash: hash}
				cmd.Printf("%s (%s): %s %s\n", name, tool, status, dst)
			}

			// Persist the installed base content once (all targets share it) so a
			// future three-way update can retrieve it by the recorded hash.
			if _, err := agentblob.Put(homontoDir, content); err != nil {
				return err
			}

			lock.Agents[name] = agentlock.Agent{
				Source:    ag.Source,
				Version:   ag.Version,
				Mode:      mode,
				Targets:   targets,
				Installed: installed,
			}
			return lock.Save(homontoDir)
		},
	}
}

// agentsUpdateCmd re-materializes an already-installed declared local: agent
// from its current source. A copy-mode target that was locally edited since the
// last install is backed up to <dst>.bak before being overwritten (backup, not
// merge); an untouched-but-stale copy is overwritten silently, and an up-to-date
// target is left alone. The lockfile hash is refreshed to the new source.
func agentsUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <name>",
		Short: "Re-materialize an installed local agent from source (backup-safe)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfgPath, _ := cmd.Flags().GetString("config")
			cfgDir := filepath.Dir(cfgPath)
			homontoDir := filepath.Join(cfgDir, ".homonto")

			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			home, _ := os.UserHomeDir()

			ag, ok := c.Agents[name]
			if !ok {
				return fmt.Errorf("agents update: agent %q is not declared", name)
			}
			if !strings.HasPrefix(ag.Source, "local:") {
				return fmt.Errorf("agents update: only local: sources are supported yet (got %q)", ag.Source)
			}

			lock, err := agentlock.Load(homontoDir)
			if err != nil {
				return err
			}
			inst, installed := lock.Agents[name]
			if !installed {
				return fmt.Errorf("agents update: agent %q is not installed (run `homonto agents add %s`)", name, name)
			}

			srcName := strings.TrimPrefix(ag.Source, "local:")
			srcPath := filepath.Join(cfgDir, "homonto", "agents", srcName+".md")
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("agents update: source file %s: %w", srcPath, err)
			}
			hash := agentlock.HashContent(content)

			mode := ag.ModeOrDefault()
			targets := ag.TargetsOrAll()
			installedRec := map[string]agentlock.Install{}
			conflicted := false
			for _, tool := range sortedStrings(targets) {
				dir := subagentpath.Dir(tool, "user", home, "")
				dst := filepath.Join(dir, name+".md")
				prev, hadRec := inst.Installed[tool]
				var status string
				switch mode {
				case "copy":
					cur, readErr := os.ReadFile(dst)
					switch {
					case readErr == nil && agentlock.HashContent(cur) == hash:
						// On-disk already equals the source — nothing to do.
						status = "up to date"
						installedRec[tool] = agentlock.Install{Path: dst, Hash: hash}
					default:
						// The recorded BASE is the ancestor this install was last
						// materialized/merged against; retrieve it by prev.Hash. An
						// unrecorded target has no ancestor to fetch.
						var base []byte
						var baseOK bool
						if hadRec && prev.Hash != "" {
							b, ok, gerr := agentblob.Get(homontoDir, prev.Hash)
							if gerr != nil {
								return gerr
							}
							base, baseOK = b, ok
						}
						if readErr != nil || !hadRec || !baseOK {
							// FALLBACK — no usable ancestor (missing base blob, missing
							// on-disk file, or a never-recorded target). Back up any
							// existing file we would clobber UNLESS it is our own
							// untouched install, then overwrite with the source. Never
							// clobber a user's file without a .bak.
							backedUp := false
							if readErr == nil && !(hadRec && agentlock.HashContent(cur) == prev.Hash) {
								if err := fsutil.WriteAtomic(dst+".bak", cur); err != nil {
									return err
								}
								backedUp = true
							}
							if err := fsutil.WriteAtomic(dst, content); err != nil {
								return err
							}
							status = "updated"
							if backedUp {
								status = fmt.Sprintf("updated (backed up local changes to %s.bak)", dst)
							}
							installedRec[tool] = agentlock.Install{Path: dst, Hash: hash}
						} else {
							// MERGE — three-way merge the local edits (cur) and the
							// upstream (content) against their common base.
							result, conflicts := merge.Merge(base, cur, content)
							if conflicts == 0 {
								if !bytes.Equal(result, cur) {
									if agentlock.HashContent(cur) != prev.Hash {
										if err := fsutil.WriteAtomic(dst+".bak", cur); err != nil {
											return err
										}
										status = fmt.Sprintf("merged (backed up local changes to %s.bak)", dst)
									} else {
										status = "merged"
									}
									if err := fsutil.WriteAtomic(dst, result); err != nil {
										return err
									}
								} else {
									status = "up to date"
								}
								// Clean merge — advance the recorded base to the source.
								installedRec[tool] = agentlock.Install{Path: dst, Hash: hash}
							} else {
								// Conflict — leave the live dst untouched, write the
								// merged-with-markers result to a sidecar, keep the
								// prior lockfile record, and fail the command.
								if err := fsutil.WriteAtomic(dst+".merged", result); err != nil {
									return err
								}
								status = fmt.Sprintf("CONFLICT (resolve %s.merged)", dst)
								conflicted = true
								if hadRec {
									installedRec[tool] = prev
								}
							}
						}
					}
				default: // link
					if isSymlinkTo(dst, srcPath) {
						status = "up to date"
					} else {
						if _, err := link.Link(srcPath, dst, homontoDir); err != nil {
							return err
						}
						status = "updated"
					}
					installedRec[tool] = agentlock.Install{Path: dst, Hash: hash}
				}
				cmd.Printf("%s (%s): %s %s\n", name, tool, status, dst)
			}

			// Persist the installed base content once (all targets share it) so a
			// future three-way update can retrieve it by the recorded hash.
			if _, err := agentblob.Put(homontoDir, content); err != nil {
				return err
			}

			lock.Agents[name] = agentlock.Agent{
				Source:    ag.Source,
				Version:   ag.Version,
				Mode:      mode,
				Targets:   targets,
				Installed: installedRec,
			}
			if err := lock.Save(homontoDir); err != nil {
				return err
			}
			if conflicted {
				return fmt.Errorf("agents update: %q has merge conflict(s); resolve the .merged file(s) and re-run", name)
			}
			return nil
		},
	}
}

// isSymlinkTo reports whether dst is a symlink whose target is exactly src.
func isSymlinkTo(dst, src string) bool {
	fi, err := os.Lstat(dst)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(dst)
	return err == nil && target == src
}
