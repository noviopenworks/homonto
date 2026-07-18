package tocli

import (
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/tostate"
	"github.com/spf13/cobra"
)

// newCmd builds "to new <change-name>": it enforces toFramework.Gate(dir) and
// toFramework.ValidChangeName before scaffolding a change directory with
// to-state.yaml (phase plan) and an empty plan.md, and performs no writes if
// either check fails or the change already exists (active or archived).
func newCmd() *cobra.Command {
	var (
		dir      string
		jsonMode bool
	)

	cmd := &cobra.Command{
		Use:   "new <change-name>",
		Short: "Create a new change (phase plan), if the to framework is installed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(cmd, dir, args[0], jsonMode)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to create the change in")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "emit the result as JSON")
	return cmd
}

func runNew(cmd *cobra.Command, root, name string, jsonMode bool) error {
	if err := toFramework.Gate(root); err != nil {
		return err
	}
	if err := toFramework.ValidChangeName(name); err != nil {
		return err
	}
	unlock, err := lock(root)
	if err != nil {
		return err
	}
	defer unlock()

	// Only an ACTIVE change blocks the name: archive dirs are date-prefixed,
	// so a finished change frees its name for reuse (recurring chores).
	if _, err := os.Stat(changeDir(root, name)); err == nil {
		return fmt.Errorf("to new: change %q already exists at %s", name, changeDir(root, name))
	}

	if err := os.MkdirAll(changeDir(root, name), 0o755); err != nil {
		return fmt.Errorf("to new: creating %s: %w", changeDir(root, name), err)
	}

	st := tostate.State{
		Change:  name,
		Phase:   tostate.PhasePlan,
		Created: todayFn(),
	}
	if err := tostate.Save(statePath(root, name), st); err != nil {
		return fmt.Errorf("to new: %w", err)
	}
	if err := os.WriteFile(planPath(root, name), []byte{}, 0o644); err != nil {
		return fmt.Errorf("to new: creating %s: %w", planPath(root, name), err)
	}

	if jsonMode {
		return printJSON(cmd, map[string]any{
			"change": name,
			"phase":  st.Phase,
			"dir":    changeDir(root, name),
			"files":  []string{statePath(root, name), planPath(root, name)},
		})
	}
	cmd.Printf("created change %q at %s\n", name, changeDir(root, name))
	cmd.Printf("  %s\n", statePath(root, name))
	cmd.Printf("  %s\n", planPath(root, name))
	return nil
}
