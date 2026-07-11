package ontocli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

// seedClose writes onto-state.yaml for name at phase "close" with the given
// deps, plus every artifact ontostate.RequiredArtifacts("close") names
// (proposal.md, tasks.md, design.md, plan.md, verification.md), each with
// placeholder content. It does not commit; callers commit explicitly so
// "clean" vs "dirty" cases are under test control.
func seedClose(t *testing.T, root, name string, deps []string) {
	t.Helper()
	changeDir := filepath.Join(root, "docs", "changes", name)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("seedClose: creating %s: %v", changeDir, err)
	}

	st := ontostate.State{Change: name, Workflow: "full", Phase: "close", Created: "2026-07-10", Deps: deps}
	if err := ontostate.Save(filepath.Join(changeDir, "onto-state.yaml"), st); err != nil {
		t.Fatalf("seedClose: saving state: %v", err)
	}

	for _, f := range ontostate.RequiredArtifacts("close") {
		if f == "onto-state.yaml" {
			continue
		}
		writeFile(t, filepath.Join(changeDir, f), "")
	}
}

// TestCloseCommand_Success verifies the happy path: a "close"-phase change
// with no deps, in a clean worktree, is archived into
// docs/changes/archive/<date>-<name>/ with Archived==true and Phase
// unchanged, and the original change directory is gone.
func TestCloseCommand_Success(t *testing.T) {
	dir := prepWorkspace(t)
	seedClose(t, dir, "demo", nil)
	commitAll(t, dir, "seed change")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"close", "demo", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	archiveDir := filepath.Join(dir, "docs", "changes", "archive", time.Now().Format("2006-01-02")+"-demo")
	st, err := ontostate.Load(filepath.Join(archiveDir, "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading archived onto-state.yaml: %v", err)
	}
	if !st.Archived {
		t.Errorf("st.Archived = false, want true")
	}
	if st.Phase != "close" {
		t.Errorf("st.Phase = %q, want %q", st.Phase, "close")
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "demo")); !os.IsNotExist(err) {
		t.Errorf("original change dir stat err = %v, want IsNotExist", err)
	}
}

// TestCloseCommand_NonClosePhaseRefused verifies that a change not yet at
// "close" is refused and left in place.
func TestCloseCommand_NonClosePhaseRefused(t *testing.T) {
	dir := prepWorkspace(t)
	seedChange(t, dir, "demo", "build")
	commitAll(t, dir, "seed change")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"close", "demo", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "demo")); err != nil {
		t.Errorf("change dir should still exist, stat err = %v", err)
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "demo", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading in-place onto-state.yaml: %v", err)
	}
	if st.Archived {
		t.Errorf("st.Archived = true after refusal, want false")
	}
}

// TestCloseCommand_UnresolvedDepRefused verifies that a change with an
// unarchived dependency is refused, naming the dependency, and not moved.
func TestCloseCommand_UnresolvedDepRefused(t *testing.T) {
	dir := prepWorkspace(t)
	seedClose(t, dir, "demo", []string{"missing"})
	commitAll(t, dir, "seed change")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"close", "demo", "--dir", dir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute() = nil, want error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("execute() error = %q, want it to mention %q", err.Error(), "missing")
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "demo")); err != nil {
		t.Errorf("change dir should still exist, stat err = %v", err)
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "demo", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading in-place onto-state.yaml: %v", err)
	}
	if st.Archived {
		t.Errorf("st.Archived = true after refusal, want false")
	}
}

// TestCloseCommand_DirtyWorktreeRefused verifies that an uncommitted change
// in the worktree blocks close and leaves the change directory in place.
func TestCloseCommand_DirtyWorktreeRefused(t *testing.T) {
	dir := prepWorkspace(t)
	seedClose(t, dir, "demo", nil)
	commitAll(t, dir, "seed change")
	writeFile(t, filepath.Join(dir, "docs", "changes", "demo", "scratch.txt"), "dirty\n")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"close", "demo", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "demo")); err != nil {
		t.Errorf("change dir should still exist, stat err = %v", err)
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "demo", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading in-place onto-state.yaml: %v", err)
	}
	if st.Archived {
		t.Errorf("st.Archived = true after refusal, want false")
	}
}

// TestCloseCommand_ArchiveTargetExistsRefused verifies no-clobber: if the
// dated archive target already exists, close is refused and the change
// directory is left in place.
func TestCloseCommand_ArchiveTargetExistsRefused(t *testing.T) {
	dir := prepWorkspace(t)
	seedClose(t, dir, "demo", nil)
	commitAll(t, dir, "seed change")

	archiveDir := filepath.Join(dir, "docs", "changes", "archive", time.Now().Format("2006-01-02")+"-demo")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("pre-creating archive target: %v", err)
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"close", "demo", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "demo")); err != nil {
		t.Errorf("change dir should still exist, stat err = %v", err)
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "demo", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading in-place onto-state.yaml: %v", err)
	}
	if st.Archived {
		t.Errorf("st.Archived = true after refusal, want false")
	}
}
