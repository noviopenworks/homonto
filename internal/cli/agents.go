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
	"github.com/noviopenworks/homonto/internal/catalog"
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

				// source drift: resolve the declared source (local: or builtin:)
				// and compare against the recorded install base hash.
				srcContent, rerr := resolveAgentSource(ag, cfgDir)
				switch {
				case rerr != nil:
					findings = append(findings, fmt.Sprintf("%s: source unresolved: %v", name, rerr))
				case len(inst.Installed) > 0:
					// Every target records the same content hash at install, so
					// compare against the first recorded target's hash (sorted for
					// determinism).
					if agentlock.HashContent(srcContent) != firstRecordedHash(inst.Installed) {
						findings = append(findings, fmt.Sprintf("%s: source changed since install (re-run `homonto agents add %s`)", name, name))
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
			if ag.ModeOrDefault() == "link" && strings.HasPrefix(ag.Source, "builtin:") {
				return fmt.Errorf("agents add: %q uses builtin: with link mode, but builtin sources have no local path to link; use mode=copy", name)
			}
			content, err := resolveAgentSource(ag, cfgDir)
			if err != nil {
				return fmt.Errorf("agents add: %w", err)
			}
			// srcPath is the local source path used by link mode; link only runs
			// for local: sources (builtin: + link is rejected above).
			srcPath := filepath.Join(cfgDir, "homonto", "agents", strings.TrimPrefix(ag.Source, "local:")+".md")
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
	var all bool
	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Re-materialize an installed local agent from source (backup-safe); --all does every agent",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			cfgDir := filepath.Dir(cfgPath)
			homontoDir := filepath.Join(cfgDir, ".homonto")

			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			home, _ := os.UserHomeDir()
			lock, err := agentlock.Load(homontoDir)
			if err != nil {
				return err
			}

			if all && len(args) > 0 {
				return fmt.Errorf("agents update: cannot combine --all with an agent name")
			}
			if !all && len(args) != 1 {
				return fmt.Errorf("agents update: provide an agent name, or use --all")
			}

			if !all {
				conflicted, err := runAgentUpdate(cmd, args[0], c, lock, cfgDir, homontoDir, home)
				if err != nil {
					return err
				}
				if err := lock.Save(homontoDir); err != nil {
					return err
				}
				if conflicted {
					return fmt.Errorf("agents update: %q has merge conflict(s); resolve the .merged file(s) and re-run", args[0])
				}
				return nil
			}

			// --all: merge every installed agent, isolating per-agent errors.
			anyConflict := false
			hadError := false
			processed, conflicted, skipped, errored := 0, 0, 0, 0
			for _, name := range sortedKeysAgents(lock.Agents) {
				if _, ok := c.Agents[name]; !ok {
					cmd.Printf("%s: skipped (no longer declared)\n", name)
					skipped++
					continue
				}
				conf, uerr := runAgentUpdate(cmd, name, c, lock, cfgDir, homontoDir, home)
				if uerr != nil {
					cmd.Printf("%s: error: %v\n", name, uerr)
					hadError = true
					errored++
					continue
				}
				processed++
				if conf {
					anyConflict = true
					conflicted++
				}
			}
			if err := lock.Save(homontoDir); err != nil {
				return err
			}
			cmd.Printf("agents update --all: %d processed, %d conflicted, %d skipped, %d errored\n", processed, conflicted, skipped, errored)
			if anyConflict || hadError {
				return fmt.Errorf("agents update --all: one or more agents had conflicts or errors")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "update every installed agent")
	return cmd
}

// runAgentUpdate re-materializes a single installed declared local: agent from
// its current source, performing the per-target three-way merge and mutating
// lock.Agents[name] in place (conflicted targets keep their prior record). It
// prints per-target statuses and reports whether any target conflicted, but does
// NOT persist the lock — the caller saves once. A hard per-agent problem
// (undeclared, non-local source, missing/unreadable source, or an IO error) is
// returned as err.
func runAgentUpdate(cmd *cobra.Command, name string, c *config.Config, lock *agentlock.Lock, cfgDir, homontoDir, home string) (conflicted bool, err error) {
	ag, ok := c.Agents[name]
	if !ok {
		return false, fmt.Errorf("agents update: agent %q is not declared", name)
	}
	if ag.ModeOrDefault() == "link" && strings.HasPrefix(ag.Source, "builtin:") {
		return false, fmt.Errorf("agents update: %q uses builtin: with link mode, but builtin sources have no local path to link; use mode=copy", name)
	}

	inst, installed := lock.Agents[name]
	if !installed {
		return false, fmt.Errorf("agents update: agent %q is not installed (run `homonto agents add %s`)", name, name)
	}

	content, err := resolveAgentSource(ag, cfgDir)
	if err != nil {
		return false, fmt.Errorf("agents update: %w", err)
	}
	// srcPath is the local source path used by link mode; link only runs for
	// local: sources (builtin: + link is rejected above).
	srcPath := filepath.Join(cfgDir, "homonto", "agents", strings.TrimPrefix(ag.Source, "local:")+".md")
	hash := agentlock.HashContent(content)

	mode := ag.ModeOrDefault()
	targets := ag.TargetsOrAll()
	installedRec := map[string]agentlock.Install{}
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
						return false, gerr
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
							return false, err
						}
						backedUp = true
					}
					if err := fsutil.WriteAtomic(dst, content); err != nil {
						return false, err
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
									return false, err
								}
								status = fmt.Sprintf("merged (backed up local changes to %s.bak)", dst)
							} else {
								status = "merged"
							}
							if err := fsutil.WriteAtomic(dst, result); err != nil {
								return false, err
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
							return false, err
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
					return false, err
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
		return false, err
	}

	lock.Agents[name] = agentlock.Agent{
		Source:    ag.Source,
		Version:   ag.Version,
		Mode:      mode,
		Targets:   targets,
		Installed: installedRec,
	}
	return conflicted, nil
}

// resolveAgentSource resolves a declared agent's source to its content:
// local:<x> reads homonto/agents/<x>.md under the config dir; builtin:<x> reads
// the embedded catalog's curated agent content by name (unknown name is an
// error); any other scheme is not yet supported (remote deferred).
func resolveAgentSource(ag config.Agent, cfgDir string) ([]byte, error) {
	switch {
	case strings.HasPrefix(ag.Source, "local:"):
		p := filepath.Join(cfgDir, "homonto", "agents", strings.TrimPrefix(ag.Source, "local:")+".md")
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("source file %s: %w", p, err)
		}
		return b, nil
	case strings.HasPrefix(ag.Source, "builtin:"):
		name := strings.TrimPrefix(ag.Source, "builtin:")
		cat, err := catalog.New()
		if err != nil {
			return nil, err
		}
		b, ok, err := cat.SubagentContent(name)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("unknown builtin agent %q", name)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported agent source %q (remote sources are not yet supported)", ag.Source)
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
