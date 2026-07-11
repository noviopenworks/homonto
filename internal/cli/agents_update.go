package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	mode, err := agentMode(name, ag)
	if err != nil {
		return false, fmt.Errorf("agents update: %w", err)
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
	// local: sources (builtin: is copy-only).
	srcPath := filepath.Join(cfgDir, "homonto", "agents", strings.TrimPrefix(ag.Source, "local:")+".md")
	hash := agentlock.HashContent(content)
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

	// Carry forward install records for targets that were removed from the
	// config: the file may still be on disk, so we must not forget we own it.
	// Rebuilding installedRec from the declared targets alone would silently drop
	// that ownership; a later `agents prune` reconciles these de-declared targets
	// (removing both the file and the record).
	declaredSet := make(map[string]bool, len(targets))
	for _, t := range targets {
		declaredSet[t] = true
	}
	for tool, rec := range inst.Installed {
		if !declaredSet[tool] {
			installedRec[tool] = rec
		}
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
