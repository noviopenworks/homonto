package cli

import (
	"bufio"
	"fmt"
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
			e, err := engine.Build(cfgPath, home, "homonto")
			if err != nil {
				return err
			}
			sets, err := e.Plan()
			if err != nil {
				return err
			}
			for _, w := range e.Warnings {
				cmd.Println("warn:", w)
			}
			// A skipped adapter means one tool was never written. Apply still
			// proceeds for the healthy tools, but the run must exit non-zero so
			// automation notices (plan/status keep exit 0 with warnings).
			skipped := func() error {
				if len(e.Warnings) == 0 {
					return nil
				}
				return fmt.Errorf("completed with skipped adapters: %s", strings.Join(e.Warnings, "; "))
			}
			// Three-way flow. Adopt is a state-only reconciliation that renders no
			// line, so it is invisible to HasChanges: (1) nothing at all → up to
			// date; (2) adoptions but no visible change → reconcile silently, no
			// diff and no prompt; (3) visible changes → render, prompt, apply
			// (any adoptions ride along inside the same Apply).
			if !plan.HasChanges(sets) {
				if !plan.HasAdoptions(sets) {
					cmd.Println("No changes. Everything up to date.")
					return skipped()
				}
				n := 0
				for _, s := range sets {
					for _, c := range s.Changes {
						if c.Action == "adopt" {
							n++
						}
					}
				}
				if err := e.Apply(sets); err != nil {
					return err
				}
				cmd.Printf("Reconciled %d pre-existing resource(s) into state.\n", n)
				return skipped()
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
			return skipped()
		},
	}
	cmd.Flags().Bool("yes", false, "skip confirmation")
	return cmd
}
