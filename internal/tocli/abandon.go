package tocli

import (
	"fmt"
	"time"

	"github.com/noviopenworks/homonto/internal/tostate"
	"github.com/spf13/cobra"
)

// abandonCmd builds "to abandon <change-name>": the terminal exit without
// done. It works from any non-terminal phase and archives the change so the
// active listing stays clean.
func abandonCmd() *cobra.Command {
	var (
		dir      string
		jsonMode bool
	)

	cmd := &cobra.Command{
		Use:   "abandon <change-name>",
		Short: "Abandon a change (terminal) and archive it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAbandon(cmd, dir, args[0], jsonMode)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "emit the result as JSON")
	return cmd
}

func runAbandon(cmd *cobra.Command, root, name string, jsonMode bool) error {
	if err := gate(root); err != nil {
		return err
	}
	st, err := loadChange(root, name)
	if err != nil {
		return err
	}
	if st.Terminal() {
		return fmt.Errorf("to abandon: change %q is %s, which is terminal", name, st.Phase)
	}

	st.Phase = tostate.PhaseAbandoned
	st.Finished = time.Now().Format("2006-01-02")
	if err := tostate.Save(statePath(root, name), st); err != nil {
		return fmt.Errorf("to abandon: %w", err)
	}

	dest, err := archive(root, name)
	if err != nil {
		return err
	}

	if jsonMode {
		return printJSON(cmd, map[string]any{"change": name, "phase": st.Phase, "archived": dest})
	}
	cmd.Printf("change %q abandoned, archived at %s\n", name, dest)
	return nil
}
