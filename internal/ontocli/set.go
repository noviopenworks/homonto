package ontocli

import (
	"fmt"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// runTransition loads the change via LoadChange (so migration + dual-legacy
// conflict detection apply), lets apply validate+mutate the state, re-validates
// the whole state, and saves. It gates on gate(root) and validChangeName, and
// writes nothing if any step fails.
func runTransition(cmd *cobra.Command, root, name string, apply func(*ontostate.State) error) error {
	if err := gate(root); err != nil {
		return err
	}
	if err := validChangeName(name); err != nil {
		return err
	}
	changeDir := filepath.Join(root, "docs", "changes", name)
	st, err := ontostate.LoadChange(changeDir)
	if err != nil {
		return fmt.Errorf("onto set: loading %s: %w", changeDir, err)
	}
	if err := apply(&st); err != nil {
		return err
	}
	if err := st.Validate(); err != nil {
		return err
	}
	if err := ontostate.Save(filepath.Join(changeDir, "onto-state.yaml"), st); err != nil {
		return fmt.Errorf("onto set: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: updated\n", name)
	return nil
}

// enumSetterCmd builds a `set <field> <change> <value>` subcommand that accepts
// only members of allowed and applies set() to the loaded state.
func enumSetterCmd(field string, allowed []string, set func(*ontostate.State, string)) *cobra.Command {
	var dir string
	allowedSet := map[string]bool{}
	for _, v := range allowed {
		allowedSet[v] = true
	}
	cmd := &cobra.Command{
		Use:   field + " <change> <value>",
		Short: "Set the " + field + " field of a change",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, value := args[0], args[1]
			return runTransition(cmd, dir, name, func(st *ontostate.State) error {
				if !allowedSet[value] {
					return fmt.Errorf("onto set %s: %q is not one of %v", field, value, allowed)
				}
				set(st, value)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	return cmd
}

// setCmd builds the "onto set" parent with one semantic subcommand per gated
// field. Each subcommand owns its field's allowed set.
func setCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a gated state field of an active change",
	}
	cmd.AddCommand(enumSetterCmd("isolation", []string{"branch", "worktree"},
		func(s *ontostate.State, v string) { s.Isolation = v }))
	cmd.AddCommand(enumSetterCmd("build-mode", []string{"direct", "subagent"},
		func(s *ontostate.State, v string) { s.BuildMode = v }))
	cmd.AddCommand(enumSetterCmd("tdd-mode", []string{"tdd", "direct"},
		func(s *ontostate.State, v string) { s.TDDMode = v }))
	cmd.AddCommand(enumSetterCmd("verify-scale", []string{"light", "full"},
		func(s *ontostate.State, v string) { s.Verify.Scale = v }))
	cmd.AddCommand(enumSetterCmd("verify-result", []string{"pending", "pass", "fail"},
		func(s *ontostate.State, v string) { s.Verify.Result = v }))
	return cmd
}
