package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/engine"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/spf13/cobra"
)

func planCmd() *cobra.Command {
	var (
		output   string
		exitFlag bool
	)
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show what apply would change",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("plan: --output %q is not one of text|json", output)
			}
			cfgPath, _ := cmd.Flags().GetString("config")
			home, _ := os.UserHomeDir()
			e, err := engine.Build(cfgPath, home, "homonto")
			if err != nil {
				return err
			}
			e.HomontoVersion = Version
			sets, err := e.Plan()
			if err != nil {
				return err
			}
			// A digest-only remote repin is invisible to the symlink plan but
			// still a pending change; surface it here too (F6).
			repins, err := e.PendingRemoteRepins()
			if err != nil {
				return err
			}
			// So is stale catalog content: a catalog file's symlink target is
			// name-based, so a re-render (model route changed) or re-extract
			// (framework content changed, rendered file deleted) moves no
			// projected value. Apply acts on it — plan must say so, or
			// automation gating apply on plan's exit code never repairs it.
			catalogStale := e.CatalogNeedsMaterialize()
			if exitFlag {
				setExitCode(planExitCode(plan.HasChanges(sets), len(repins), catalogStale))
			}
			if output == "json" {
				return planJSON(cmd, sets, repins, e.Warnings)
			}
			for _, w := range e.Warnings {
				cmd.Println("warn:", w)
			}
			if !plan.HasChanges(sets) && len(repins) == 0 {
				if err := coverageComplete(e.Warnings); err != nil {
					return err
				}
				if catalogStale {
					cmd.Println("No projection changes; catalog re-materialization pending (run `homonto apply`).")
					return nil
				}
				cmd.Println("No changes. Everything up to date.")
				return nil
			}
			cmd.Print(plan.Render(sets))
			if len(repins) > 0 {
				cmd.Print(renderRepins(repins))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&output, "output", "text", "output format: text or json")
	cmd.Flags().BoolVar(&exitFlag, "exit-code", false, "exit 2 when changes are pending (opt-in taxonomy)")
	return cmd
}

// planJSON emits the plan as a conservative machine-readable object: per-tool
// visible changes as {action, key} only (never Old/New, which can carry
// unresolved secret tokens), pending remote repins by name, and warnings.
func planJSON(cmd *cobra.Command, sets []adapter.ChangeSet, repins []engine.RemoteRepin, warnings []string) error {
	type changeJSON struct {
		Action string `json:"action"`
		Key    string `json:"key"`
	}
	type setJSON struct {
		Tool    string       `json:"tool"`
		Changes []changeJSON `json:"changes"`
	}
	setsOut := []setJSON{}
	for _, s := range sets {
		var cs []changeJSON
		for _, c := range s.Changes {
			if c.Action == "noop" {
				continue
			}
			cs = append(cs, changeJSON{Action: string(c.Action), Key: c.Key})
		}
		if len(cs) > 0 {
			setsOut = append(setsOut, setJSON{Tool: s.Tool, Changes: cs})
		}
	}
	repinsOut := []struct {
		Name string `json:"name"`
	}{}
	for _, r := range repins {
		repinsOut = append(repinsOut, struct {
			Name string `json:"name"`
		}{Name: r.Name})
	}
	if warnings == nil {
		warnings = []string{}
	}
	b, err := json.MarshalIndent(struct {
		Changes []setJSON `json:"changes"`
		Repins  []struct {
			Name string `json:"name"`
		} `json:"repins"`
		Warnings []string `json:"warnings"`
	}{Changes: setsOut, Repins: repinsOut, Warnings: warnings}, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(b))
	return nil
}
