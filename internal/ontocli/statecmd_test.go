package ontocli

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestStateJSON_EmitsFullStateAndDerivedPhase(t *testing.T) {
	root := t.TempDir() // read command needs no framework install
	writeFile(t, filepath.Join(root, "docs", "changes", "c", "onto-state.yaml"),
		"schema_version: 1\nchange: c\nworkflow: full\nphase: build\nisolation: worktree\n")

	before := treeSnapshot(t, root)

	out, err := runOnto(t, "state", "c", "--json", "--dir", root)
	if err != nil {
		t.Fatalf("state --json: %v", err)
	}

	var got struct {
		Change       string `json:"change"`
		Phase        string `json:"phase"`
		Isolation    string `json:"isolation"`
		DerivedPhase string `json:"derived_phase"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if got.Change != "c" || got.Isolation != "worktree" {
		t.Errorf("state = %+v, want change=c isolation=worktree", got)
	}
	if got.DerivedPhase != "build" {
		t.Errorf("derived_phase = %q, want build", got.DerivedPhase)
	}

	after := treeSnapshot(t, root)
	if len(before) != len(after) {
		t.Errorf("state --json mutated the tree: before=%d after=%d files", len(before), len(after))
	}
}
