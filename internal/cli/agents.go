package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/link"
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
					if inst.Mode == "copy" {
						b, rerr := os.ReadFile(ti.Path)
						if rerr != nil {
							findings = append(findings, fmt.Sprintf("%s (%s): installed file unreadable: %s", name, tool, ti.Path))
						} else if agentlock.HashContent(b) != ti.Hash {
							findings = append(findings, fmt.Sprintf("%s (%s): modified on disk: %s", name, tool, ti.Path))
						}
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

// isSymlinkTo reports whether dst is a symlink whose target is exactly src.
func isSymlinkTo(dst, src string) bool {
	fi, err := os.Lstat(dst)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(dst)
	return err == nil && target == src
}
