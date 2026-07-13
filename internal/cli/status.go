package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show config drift since last apply",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("status: --output %q is not one of text|json", output)
			}
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
			if output == "json" {
				payload := struct {
					Drift    []string `json:"drift"`
					Pending  int      `json:"pending"`
					Warnings []string `json:"warnings"`
				}{Drift: drift, Pending: pending, Warnings: e.Warnings}
				if payload.Drift == nil {
					payload.Drift = []string{}
				}
				if payload.Warnings == nil {
					payload.Warnings = []string{}
				}
				b, merr := json.MarshalIndent(payload, "", "  ")
				if merr != nil {
					return merr
				}
				cmd.Println(string(b))
				return nil
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
				if err := coverageComplete(e.Warnings); err != nil {
					return err
				}
				cmd.Println("No drift.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&output, "output", "text", "output format: text or json")
	return cmd
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
