package ontocli

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// handoffCmd builds "onto handoff <change> [--write]": a compact, recovery-
// oriented context pack for a change — identity, phase, deps, the pending gate,
// and pointers to the workspace artifacts with a content hash. After a context
// compaction the derivation recovers the *phase*; the handoff recovers *what the
// change is about* so a fresh agent can continue without re-reading everything.
// --write saves it under the workspace so it survives the session.
func handoffCmd() *cobra.Command {
	var (
		dir     string
		doWrite bool
	)
	cmd := &cobra.Command{
		Use:   "handoff <change>",
		Short: "Emit a compact recovery context pack for a change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := ontoFramework.ValidChangeName(name); err != nil {
				return err
			}
			changeDir := filepath.Join(dir, "docs", "changes", name)
			st, err := ontostate.LoadChange(changeDir)
			if err != nil {
				return err
			}
			// The phase feeds the output filename; a malformed or traversal-
			// carrying value must never reach filepath.Join. Reject any value
			// outside the recognized phase set BEFORE --write builds a path from
			// it (F6). LoadChange does not Validate, so this is the gate.
			if !ontostate.ValidPhase(st.Phase) {
				return fmt.Errorf("onto handoff: %q has an unknown phase %q; refusing to build a handoff path from it", name, st.Phase)
			}
			pack, err := buildHandoff(name, changeDir, st)
			if err != nil {
				return err
			}
			if doWrite {
				out := filepath.Join(changeDir, ".onto", "handoff", st.Phase+"-context.md")
				// WriteControlPlane refuses a symlink at the destination and uses a
				// unique temp name, so a planted link cannot redirect the write
				// outside the workspace (F6). A regular existing file is replaced
				// atomically as before.
				if err := fsutil.WriteControlPlane(out, []byte(pack), 0o644); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", out)
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), pack)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root containing the change")
	cmd.Flags().BoolVar(&doWrite, "write", false, "save the pack under docs/changes/<name>/.onto/handoff/")
	return cmd
}

// handoffArtifacts are the workspace files the pack summarizes and hashes, in
// the order a reader wants them.
var handoffArtifacts = []string{"proposal.md", "notes.md", "design.md", "plan.md", "tasks.md", "verification.md"}

func buildHandoff(name, changeDir string, st ontostate.State) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# onto handoff: %s\n\n", name)
	fmt.Fprintf(&b, "- **id**: %s\n- **workflow**: %s\n- **phase**: %s\n", nonEmpty(st.ID, "(none)"), nonEmpty(st.Workflow, "full"), st.Phase)
	if len(st.Deps) > 0 {
		fmt.Fprintf(&b, "- **deps**: %s\n", strings.Join(st.Deps, ", "))
	}
	if st.BaseRef != "" {
		fmt.Fprintf(&b, "- **base_ref**: %s\n", st.BaseRef)
	}

	if gates := pendingGates(name, st); len(gates) > 0 {
		b.WriteString("\n## Pending decision\n\n")
		for _, g := range gates {
			fmt.Fprintf(&b, "- **%s** — %s (`%s`)\n", g.Header, g.Question, g.SetCommand)
		}
	}

	b.WriteString("\n## Artifacts (present, with a content hash for staleness)\n\n")
	h := sha256.New()
	any := false
	for _, f := range handoffArtifacts {
		data, err := os.ReadFile(filepath.Join(changeDir, f))
		if err != nil {
			continue
		}
		any = true
		h.Write([]byte(f))
		h.Write(data)
		fmt.Fprintf(&b, "- `%s` — %s\n", f, firstMeaningfulLine(data))
	}
	if !any {
		b.WriteString("- (no artifacts yet)\n")
	}
	fmt.Fprintf(&b, "\n**artifacts-hash**: sha256:%x\n", h.Sum(nil))
	b.WriteString("\nRecover the phase from file state (the onto dispatcher's derivation); this pack is the *content* recovery. Re-read an artifact above if its excerpt is not enough.\n")
	return b.String(), nil
}

func nonEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// firstMeaningfulLine returns the first non-blank, non-heading, non-marker line
// of an artifact as a one-line excerpt.
func firstMeaningfulLine(data []byte) string {
	for _, ln := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "---") {
			continue
		}
		if len(t) > 100 {
			t = t[:100] + "…"
		}
		return t
	}
	return "(empty)"
}
