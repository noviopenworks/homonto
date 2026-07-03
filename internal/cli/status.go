package cli

import (
	"os"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show config drift since last apply",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "content")
			if err != nil {
				return err
			}
			lines, err := e.Drift()
			if err != nil {
				return err
			}
			if len(lines) == 0 {
				cmd.Println("No drift.")
				return nil
			}
			for _, l := range lines {
				cmd.Println(l)
			}
			return nil
		},
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "content")
			if err != nil {
				return err
			}
			for _, l := range e.Doctor() {
				cmd.Println(l)
			}
			return nil
		},
	}
}
