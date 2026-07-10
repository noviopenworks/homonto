package ontocli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// closeCmd builds the "onto close <change>" subcommand: it enforces
// gate(dir), validates the change name, and only then attempts to archive a
// change already at the terminal "close" phase. It writes nothing and moves
// nothing unless every precondition below passes.
func closeCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "close <change>",
		Short: "Archive a change that has reached the close phase, if all gates pass",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClose(cmd, dir, args[0])
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}

// runClose enforces, in order: gate(root); validChangeName(name); that
// docs/changes/<name>/onto-state.yaml loads; that its phase is "close" (the
// terminal phase reached via repeated "onto advance"); that every
// dependency named in st.Deps has already been archived
// (ontostate.DepsResolved); that the worktree is clean and that cleanliness
// is determinable; and that the dated archive target does not already
// exist (no-clobber). Only once all of these pass does it mark the state
// Archived, save it, and move the change directory into
// docs/changes/archive/<date>-<name>/.
func runClose(cmd *cobra.Command, root, name string) error {
	if err := gate(root); err != nil {
		return err
	}

	if err := validChangeName(name); err != nil {
		return err
	}

	changeDir := filepath.Join(root, "docs", "changes", name)
	statePath := filepath.Join(changeDir, "onto-state.yaml")

	st, err := ontostate.Load(statePath)
	if err != nil {
		return fmt.Errorf("onto close: loading %s: %w", statePath, err)
	}

	if st.Phase != "close" {
		return fmt.Errorf("onto close: change %q is at phase %q; run `onto advance` until it reaches close", name, st.Phase)
	}

	unresolved := ontostate.DepsResolved(root, st.Deps)
	if len(unresolved) > 0 {
		return fmt.Errorf("onto close: unresolved dependencies: %v", unresolved)
	}

	dirty, determinable := worktreeDirty(root)
	if dirty {
		return fmt.Errorf("onto close: dirty worktree blocks close")
	}
	if !determinable {
		return fmt.Errorf("onto close: cannot verify worktree is clean; refusing close")
	}

	archiveDir := filepath.Join(root, "docs", "changes", "archive", time.Now().Format("2006-01-02")+"-"+name)
	if _, err := os.Stat(archiveDir); err == nil {
		return fmt.Errorf("onto close: archive target already exists: %s", archiveDir)
	}

	st.Archived = true
	if err := ontostate.Save(statePath, st); err != nil {
		return fmt.Errorf("onto close: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "docs", "changes", "archive"), 0o755); err != nil {
		return fmt.Errorf("onto close: creating archive directory: %w", err)
	}
	if err := os.Rename(changeDir, archiveDir); err != nil {
		return fmt.Errorf("onto close: moving %s to %s: %w", changeDir, archiveDir, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s: archived to %s\n", name, archiveDir)
	return nil
}
