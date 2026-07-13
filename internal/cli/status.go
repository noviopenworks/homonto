package cli

import (
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show config drift since last apply",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "homonto")
			if err != nil {
				return err
			}
			drift, pending, err := e.Status()
			if err != nil {
				return err
			}
			for _, w := range e.Warnings {
				cmd.Println("warn:", w)
			}
			for _, l := range drift {
				cmd.Println(l)
			}
			if pending > 0 {
				cmd.Println(fmt.Sprintf("%d config change(s) awaiting apply (run `homonto apply`)", pending))
			}
			if len(drift) == 0 && pending == 0 {
				cmd.Println("No drift.")
			}
			return nil
		},
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "homonto")
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
