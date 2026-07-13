package ontostate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadChange_BothLegacyAgree_MergesObserved(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "onto-state.yaml", "change: c\nworkflow: full\nphase: build\narchived: false\n")
	writeFixture(t, dir, "state.yaml", "change: c\nworkflow: full\nphase: build\narchived: false\nmetrics:\n  phases:\n    open: \"2026-07-10\"\n  tasks_total: 3\n")

	st, err := LoadChange(dir)
	if err != nil {
		t.Fatalf("LoadChange: %v", err)
	}
	if st.Phase != "build" {
		t.Errorf("Phase = %q, want build", st.Phase)
	}
	if st.Observed.TasksTotal != 3 || st.Observed.Metrics["open"] != "2026-07-10" {
		t.Errorf("Observed not merged from skill file: %+v", st.Observed)
	}
}

func TestLoadChange_BothLegacyDisagree_IsMalformed(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "onto-state.yaml", "change: c\nphase: build\n")
	writeFixture(t, dir, "state.yaml", "change: c\nphase: verify\n") // phase disagrees

	_, err := LoadChange(dir)
	if err == nil {
		t.Fatal("LoadChange accepted disagreeing dual legacy files, want malformed error")
	}
	if !strings.Contains(err.Error(), "phase") {
		t.Errorf("error = %q, want it to name the conflicting field", err.Error())
	}
}

func TestClassify_MissingState(t *testing.T) {
	dir := t.TempDir() // directory exists, no state file
	_, class, err := Classify(dir)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if class != "missing-state" {
		t.Errorf("class = %q, want missing-state", class)
	}
}

func TestClassify_ValidAndMalformed(t *testing.T) {
	valid := t.TempDir()
	writeFixture(t, valid, "onto-state.yaml", "change: c\nphase: build\n")
	if st, class, err := Classify(valid); err != nil || class != "valid" || st.Phase != "build" {
		t.Errorf("valid case: class=%q phase=%q err=%v", class, st.Phase, err)
	}

	bad := t.TempDir()
	writeFixture(t, bad, "onto-state.yaml", "change: c\nphase: bogus\n")
	if _, class, err := Classify(bad); class != "malformed" || err == nil {
		t.Errorf("malformed case: class=%q err=%v, want malformed + error", class, err)
	}
}
