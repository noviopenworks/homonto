package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/spf13/cobra"
)

// agentsPruneCmd builds "agents prune": it removes homonto-managed agent installs
// that are no longer declared — an orphan agent (recorded but undeclared) and a
// de-declared target (a recorded target the agent no longer targets) — and drops
// their lockfile records. It touches only recorded install paths, backs a
// locally-edited file up to <path>.bak before removing it, and cleans up a
// leftover <path>.merged sidecar. --dry-run lists what would be pruned and
// changes nothing.
func agentsPruneCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove orphaned/de-declared agent installs",
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

			var actions []string
			changed := false

			// pruneFile removes a recorded install (backing up a local edit first,
			// clearing a .merged sidecar). It returns whether the install may be
			// dropped from the lockfile: true when removed, already gone, or a
			// dry-run preview; FALSE only when a required backup write failed — in
			// which case the file is KEPT (never removed without its .bak) so no
			// user edit is lost, and the record is kept too so a retry can prune it.
			pruneFile := func(ti agentlock.Install) bool {
				if _, err := os.Lstat(ti.Path); err != nil {
					return true // already gone
				}
				if dryRun {
					actions = append(actions, fmt.Sprintf("would remove %s", ti.Path))
					return true
				}
				// Back up a local edit (on-disk content differs from recorded base).
				if b, rerr := os.ReadFile(ti.Path); rerr == nil && agentlock.HashContent(b) != ti.Hash {
					if err := fsutil.WriteAtomic(ti.Path+".bak", b); err != nil {
						actions = append(actions, fmt.Sprintf("SKIPPED %s: backup to .bak failed (%v); file kept", ti.Path, err))
						return false // keep the file AND its lockfile record
					}
					actions = append(actions, fmt.Sprintf("backed up %s to %s.bak", ti.Path, ti.Path))
				}
				// Remove the optional conflict sidecar first (a missing one is
				// fine), then the install itself. A real deletion failure keeps the
				// file AND its lockfile record so a retry can prune it — ownership is
				// never dropped and a failed removal is never reported as "removed".
				if err := os.Remove(ti.Path + ".merged"); err != nil && !os.IsNotExist(err) {
					actions = append(actions, fmt.Sprintf("SKIPPED %s: could not remove %s.merged (%v); file kept", ti.Path, ti.Path, err))
					return false
				}
				if err := os.Remove(ti.Path); err != nil && !os.IsNotExist(err) {
					actions = append(actions, fmt.Sprintf("SKIPPED %s: remove failed (%v); file kept", ti.Path, err))
					return false
				}
				actions = append(actions, fmt.Sprintf("removed %s", ti.Path))
				return true
			}

			for _, name := range sortedKeysAgents(lock.Agents) {
				ag, declared := c.Agents[name]
				inst := lock.Agents[name]
				if !declared {
					// Orphan: prune every recorded target. Drop the agent record
					// only if all its targets were safely pruned.
					allPruned := true
					for _, tool := range sortedKeys(inst.Installed) {
						if !pruneFile(inst.Installed[tool]) {
							allPruned = false
						}
					}
					if allPruned {
						actions = append(actions, fmt.Sprintf("pruned orphan agent %q", name))
						if !dryRun {
							delete(lock.Agents, name)
							changed = true
						}
					}
					continue
				}
				// De-declared targets: recorded target not in ag.TargetsOrAll().
				declaredSet := map[string]bool{}
				for _, tool := range ag.TargetsOrAll() {
					declaredSet[tool] = true
				}
				for _, tool := range sortedKeys(inst.Installed) {
					if declaredSet[tool] {
						continue
					}
					if !pruneFile(inst.Installed[tool]) {
						continue // backup failed → keep the target record
					}
					actions = append(actions, fmt.Sprintf("pruned de-declared target %s of %q", tool, name))
					if !dryRun {
						delete(inst.Installed, tool)
						lock.Agents[name] = inst
						changed = true
					}
				}
			}

			if len(actions) == 0 {
				cmd.Println("nothing to prune")
				return nil
			}
			for _, a := range actions {
				cmd.Println(a)
			}
			if dryRun {
				cmd.Println("(dry run — nothing changed)")
				return nil
			}
			if changed {
				if err := lock.Save(homontoDir); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list what would be pruned without changing anything")
	return cmd
}
