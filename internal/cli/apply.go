package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/spf13/cobra"
)

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Project config into the AI tools",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			yes, _ := cmd.Flags().GetBool("yes")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "content")
			if err != nil {
				return err
			}
			sets, err := e.Plan()
			if err != nil {
				return err
			}
			if !plan.HasChanges(sets) {
				cmd.Println("No changes. Everything up to date.")
				return nil
			}
			cmd.Print(plan.Render(sets))
			if !yes {
				cmd.Print("\nApply these changes? [y/N] ")
				r := bufio.NewReader(os.Stdin)
				line, _ := r.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(line)) != "y" {
					cmd.Println("Aborted.")
					return nil
				}
			}
			if err := e.Apply(sets); err != nil {
				return err
			}
			cmd.Println("Applied.")
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation")
	return cmd
}
