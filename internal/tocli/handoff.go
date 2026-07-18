package tocli

import (
	"fmt"
	"os"
	"strings"

	"github.com/noviopenworks/homonto/internal/tostate"
	"github.com/spf13/cobra"
)

// planExcerptLines caps how much of plan.md a handoff carries: enough to
// resume, small enough to paste into a fresh context.
const planExcerptLines = 60

// handoffCmd builds "to handoff <change-name>": a compact context-recovery
// pack (identity, phase, plan excerpt, the next command) for continuing
// after a context compaction. Read-only and config-independent.
func handoffCmd() *cobra.Command {
	var (
		dir      string
		jsonMode bool
	)

	cmd := &cobra.Command{
		Use:   "handoff <change-name>",
		Short: "Print a compact recovery pack for a change (read-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHandoff(cmd, dir, args[0], jsonMode)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "emit the result as JSON")
	return cmd
}

// nextStep maps a phase to the action that continues the change.
func nextStep(name, phase string) string {
	switch phase {
	case tostate.PhasePlan:
		return fmt.Sprintf("write plan.md, then `to phase %s`", name)
	case tostate.PhaseDo:
		return fmt.Sprintf("execute plan.md; when verified, `to done %s --verified`", name)
	default:
		return "none — the change is terminal"
	}
}

func runHandoff(cmd *cobra.Command, root, name string, jsonMode bool) error {
	st, err := loadChange(root, name)
	if err != nil {
		return err
	}

	excerpt := ""
	if b, err := os.ReadFile(planPath(root, name)); err == nil {
		lines := strings.Split(string(b), "\n")
		if len(lines) > planExcerptLines {
			lines = append(lines[:planExcerptLines], "… (plan.md truncated)")
		}
		excerpt = strings.TrimRight(strings.Join(lines, "\n"), "\n")
	}

	if jsonMode {
		return printJSON(cmd, map[string]any{
			"change": name,
			"state":  st,
			"plan":   excerpt,
			"next":   nextStep(name, st.Phase),
		})
	}

	cmd.Printf("change: %s\nphase: %s\ncreated: %s\nnext: %s\n", name, st.Phase, st.Created, nextStep(name, st.Phase))
	if excerpt != "" {
		cmd.Printf("\nplan.md:\n%s\n", excerpt)
	}
	return nil
}
