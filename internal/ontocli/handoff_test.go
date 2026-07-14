package ontocli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

func TestHandoff_ContentAndWrite(t *testing.T) {
	root := prepWorkspace(t)
	changeDir := filepath.Join(root, "docs", "changes", "c")
	ontostate.Save(filepath.Join(changeDir, "onto-state.yaml"),
		ontostate.State{Change: "c", ID: "abcd1234", Workflow: "full", Phase: "design", Deps: []string{"dep-a"}})
	writeFile(t, filepath.Join(changeDir, "proposal.md"), "# Proposal\n\nBuild the thing.\n")

	// stdout form carries identity, the pending gate, the artifact excerpt + hash.
	out, err := runOnto(t, "handoff", "c", "--dir", root)
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	for _, want := range []string{"onto handoff: c", "abcd1234", "phase**: design", "deps**: dep-a", "Pending decision", "Isolation", "Build the thing.", "artifacts-hash**: sha256:"} {
		if !strings.Contains(out, want) {
			t.Errorf("handoff missing %q:\n%s", want, out)
		}
	}

	// --write persists it under the workspace.
	if _, err := runOnto(t, "handoff", "c", "--dir", root, "--write"); err != nil {
		t.Fatalf("handoff --write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(changeDir, ".onto", "handoff", "design-context.md")); err != nil {
		t.Errorf("handoff --write did not persist the pack: %v", err)
	}
}
