package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/noviopenworks/homonto/internal/applylock"
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
			// Serialize concurrent applies on the same project: two applies must
			// not plan from the same snapshot and race to a last-writer-wins
			// outcome on the state and tool files. Held from before Plan (so the
			// snapshot is stable) until the command returns.
			lock, err := applylock.Acquire(e.StateDir)
			if err != nil {
				return err
			}
			defer lock.Release()
			sets, err := e.Plan()
			if err != nil {
				return err
			}
			for _, w := range e.Warnings {
				cmd.Println("warn:", w)
			}
			// A digest-only remote repin leaves the name-based symlink plan empty
			// but WILL mutate remote content; surface and confirm it (F6).
			repins, err := e.PendingRemoteRepins()
			if err != nil {
				return err
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
					// A digest-only repin is invisible to the symlink plan but
					// mutates remote content: render it and require confirmation
					// before applying (F6), never under a "no changes" conclusion.
					if len(repins) > 0 {
						cmd.Print(renderRepins(repins))
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
					}
					// A remote resource's symlink target is name-based, so an
					// unchanged remote leaves the projection plan empty. Still run
					// apply so remotes are re-fetched, pin-verified, and
					// re-materialized (fail-closed) rather than silently serving stale
					// pinned content.
					if e.HasRemoteResources() {
						if err := e.Apply(sets); err != nil {
							return err
						}
						cmd.Println("No projection changes; remote sources verified.")
						return skipped()
					}
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
			if len(repins) > 0 {
				cmd.Print(renderRepins(repins))
			}
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

// renderRepins formats digest-only remote repins as terraform-style change
// lines so plan/apply surface them even though the symlink projection is empty.
func renderRepins(repins []engine.RemoteRepin) string {
	var b strings.Builder
	b.WriteString("remote:\n")
	for _, r := range repins {
		fmt.Fprintf(&b, "  ~ subagent.%s (repin): %s -> %s\n", r.Name, r.Old, r.New)
	}
	return b.String()
}
