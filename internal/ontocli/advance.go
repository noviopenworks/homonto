package ontocli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// worktreeDirty reports whether the git worktree rooted at root has any
// uncommitted changes (dirty), and whether that could be determined at all
// (determinable). determinable is false when git is unavailable or root is
// not inside a git repository; callers must treat that as "unknown," not as
// clean.
func worktreeDirty(root string) (dirty bool, determinable bool) {
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	if err != nil {
		return false, false
	}
	return strings.TrimSpace(string(out)) != "", true
}

// advanceCmd builds the "onto advance <change>" subcommand: it enforces
// gate(dir), validates the change name, and only then attempts a single
// gated phase transition on that change's onto-state.yaml. It writes
// nothing unless every precondition below passes.
func advanceCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "advance <change>",
		Short: "Advance a change to its next workflow phase, if all gates pass",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdvance(cmd, dir, args[0])
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}

// runAdvance enforces, in order: gate(root); validChangeName(name); that
// docs/changes/<name>/onto-state.yaml loads; that its phase has a next
// phase; that every RequiredArtifacts(st.Phase) file — the current phase's
// cumulative deliverables — is present in the change directory; that, when
// leaving "build", tasks.md has no unchecked items;
// and a worktree-dirty check that unconditionally blocks entering "close"
// (refusing when dirtiness can't even be determined) but only warns for
// every other transition. Only once all of these pass does it flip the
// phase and save onto-state.yaml.
func runAdvance(cmd *cobra.Command, root, name string) error {
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
		return fmt.Errorf("onto advance: loading %s: %w", statePath, err)
	}

	next, ok := ontostate.NextPhase(st.Phase)
	if !ok {
		return fmt.Errorf("onto advance: %q is at terminal/unknown phase %q; nothing to advance", name, st.Phase)
	}

	for _, f := range ontostate.RequiredArtifacts(st.Phase) {
		if _, statErr := os.Stat(filepath.Join(changeDir, f)); statErr != nil {
			return fmt.Errorf("onto advance: cannot leave %q: missing %s", st.Phase, f)
		}
	}

	if st.Phase == "build" {
		done, tasksErr := ontostate.TasksAllChecked(filepath.Join(changeDir, "tasks.md"))
		if tasksErr != nil {
			return tasksErr
		}
		if !done {
			return fmt.Errorf("onto advance: cannot leave build: tasks.md has unchecked items")
		}
	}

	dirty, determinable := worktreeDirty(root)
	if next == "close" {
		if dirty {
			return fmt.Errorf("onto advance: dirty worktree blocks close")
		}
		if !determinable {
			return fmt.Errorf("onto advance: cannot verify worktree is clean; refusing close")
		}
	} else if dirty {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: worktree has uncommitted changes")
	}

	old := st.Phase
	st.Phase = next
	if err := ontostate.Save(statePath, st); err != nil {
		return fmt.Errorf("onto advance: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s: %s → %s\n", name, old, next)
	return nil
}
