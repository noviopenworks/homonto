package tocli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/noviopenworks/homonto/internal/tostate"
	"github.com/spf13/cobra"
)

// Excerpt caps: a short plan is carried whole; a long plan is cut to its head
// (the goal) plus every still-unchecked task — the part a resuming session
// actually needs — rather than its first N lines (which, mid-do, are mostly
// finished history).
const (
	planExcerptLines = 60
	planHeadLines    = 20
)

// uncheckedTask matches an open checkbox task line ("- [ ]" / "* [ ]",
// indented or not) — the resume unit of the to-do loop.
var uncheckedTask = regexp.MustCompile(`^\s*[-*] \[ \] `)

// excerptPlan reduces plan.md content to a resume-sized excerpt.
func excerptPlan(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= planExcerptLines {
		return strings.TrimRight(content, "\n")
	}
	out := append([]string{}, lines[:planHeadLines]...)
	out = append(out, "… (plan.md truncated; unchecked tasks below)")
	for _, l := range lines[planHeadLines:] {
		if uncheckedTask.MatchString(l) {
			out = append(out, l)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

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
		excerpt = excerptPlan(string(b))
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
