package tocli

import (
	"fmt"
	"time"

	"github.com/noviopenworks/homonto/internal/tostate"
	"github.com/spf13/cobra"
)

// doneCmd builds "to done <change-name> --verified": mark a change done and
// archive it. --verified is REQUIRED but self-asserted — the binary records
// the checkbox, it does not observe evidence. Real verification rigor lives
// in the to-done skill (run it, paste the outcome into the change notes).
func doneCmd() *cobra.Command {
	var (
		dir      string
		jsonMode bool
		verified bool
	)

	cmd := &cobra.Command{
		Use:   "done <change-name> --verified",
		Short: "Mark a change done and archive it (requires the self-asserted --verified)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDone(cmd, dir, args[0], verified, jsonMode)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "emit the result as JSON")
	cmd.Flags().BoolVar(&verified, "verified", false, "assert that the change was verified (recorded verbatim, not checked)")
	return cmd
}

func runDone(cmd *cobra.Command, root, name string, verified, jsonMode bool) error {
	if err := gate(root); err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("to done: --verified is required; verify the change first (the to-done skill), then assert it")
	}
	st, err := loadChange(root, name)
	if err != nil {
		return err
	}

	switch st.Phase {
	case tostate.PhaseDo:
		// the one phase done may leave from
	case tostate.PhasePlan:
		return fmt.Errorf("to done: change %q is still at plan; advance it with `to phase %s` first", name, name)
	default:
		return fmt.Errorf("to done: change %q is %s, which is terminal", name, st.Phase)
	}

	st.Phase = tostate.PhaseDone
	st.Verified = true
	st.Finished = time.Now().Format("2006-01-02")
	if err := tostate.Save(statePath(root, name), st); err != nil {
		return fmt.Errorf("to done: %w", err)
	}

	dest, err := archive(root, name)
	if err != nil {
		return err
	}

	if jsonMode {
		return printJSON(cmd, map[string]any{
			"change": name, "phase": st.Phase, "verified": st.Verified, "archived": dest,
		})
	}
	cmd.Printf("change %q done, archived at %s\n", name, dest)
	return nil
}
