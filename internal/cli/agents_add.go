package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/noviopenworks/homonto/internal/agentblob"
	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/link"
	"github.com/noviopenworks/homonto/internal/subagentpath"
	"github.com/spf13/cobra"
)

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
			mode, err := agentMode(name, ag)
			if err != nil {
				return fmt.Errorf("agents add: %w", err)
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
