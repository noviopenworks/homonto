package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/spf13/cobra"
)

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
