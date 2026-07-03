package cli

import (
	"os"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/spf13/cobra"
)

func planCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show what apply would change",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
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
			return nil
		},
	}
}
