package ontocli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

// runGit runs a git subcommand in dir, failing the test on error. It is used
// only to build and manipulate the temp git repos these tests exercise —
// never to touch the real repository.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// prepWorkspace builds a temp git repo that passes gate(): a homonto.toml
// declaring [frameworks.onto] plus an applied onto catalog directory. It is
// committed so the repo starts clean, and callers control dirtiness from
// there by writing further, uncommitted files.
func prepWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "homonto.toml"), "[frameworks.onto]\nsource=\"builtin:onto\"\nscope=\"project\"\n")
	if err := os.MkdirAll(filepath.Join(dir, ".homonto", "catalog", "skills", "onto"), 0o755); err != nil {
		t.Fatalf("failed to create catalog dir: %v", err)
	}
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

// seedChange writes onto-state.yaml for name at phase, plus every artifact
// RequiredArtifacts(phase) names (empty content), and any extraFiles
// (relative to the change dir, written with placeholder content). It does
// not commit; callers commit explicitly so "clean" vs "dirty" cases are
// under test control.
func seedChange(t *testing.T, root, name, phase string, extraFiles ...string) {
	t.Helper()
	changeDir := filepath.Join(root, "docs", "changes", name)

	st := ontostate.State{Change: name, Workflow: "full", Phase: phase, Created: "2026-07-10"}
	if err := ontostate.Save(filepath.Join(changeDir, "onto-state.yaml"), st); err != nil {
		t.Fatalf("seedChange: saving state: %v", err)
	}

	for _, f := range ontostate.RequiredArtifacts(phase) {
		if f == "onto-state.yaml" {
			continue
		}
		path := filepath.Join(changeDir, f)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		writeFile(t, path, "")
	}

	for _, f := range extraFiles {
		writeFile(t, filepath.Join(changeDir, f), "")
	}
}

// commitAll stages and commits everything currently in root, leaving the
// worktree clean.
func commitAll(t *testing.T, root, msg string) {
	t.Helper()
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-m", msg)
}

// dirtyWorktree writes an untracked file into root so the worktree is no
// longer clean, without committing it.
func dirtyWorktree(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "uncommitted.txt"), "dirty\n")
}

// --- worktreeDirty ---

func TestWorktreeDirty_CleanRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	writeFile(t, filepath.Join(dir, "a.txt"), "a\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")

	dirty, determinable := worktreeDirty(dir)
	if !determinable {
		t.Fatal("worktreeDirty() determinable = false, want true")
	}
	if dirty {
		t.Error("worktreeDirty() dirty = true, want false")
	}
}

func TestWorktreeDirty_DirtyRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	writeFile(t, filepath.Join(dir, "a.txt"), "a\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")

	writeFile(t, filepath.Join(dir, "b.txt"), "b\n")

	dirty, determinable := worktreeDirty(dir)
	if !determinable {
		t.Fatal("worktreeDirty() determinable = false, want true")
	}
	if !dirty {
		t.Error("worktreeDirty() dirty = false, want true")
	}
}

func TestWorktreeDirty_NonRepo(t *testing.T) {
	dir := t.TempDir()

	_, determinable := worktreeDirty(dir)
	if determinable {
		t.Error("worktreeDirty() determinable = true, want false for a non-repo dir")
	}
}

// --- advance command ---

// TestAdvanceCommand_OpenToDesign verifies a normal, unblocked advance: an
// open change with design.md present moves to design and exits 0.
func TestAdvanceCommand_OpenToDesign(t *testing.T) {
	dir := prepWorkspace(t)
	seedChange(t, dir, "feature-x", "open", "design.md")
	commitAll(t, dir, "seed change")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"advance", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "feature-x", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading onto-state.yaml: %v", err)
	}
	if st.Phase != "design" {
		t.Errorf("st.Phase = %q, want %q", st.Phase, "design")
	}
}

// TestAdvanceCommand_MissingDesignDocRefused verifies that advancing an open
// change without design.md is refused and leaves the phase unchanged.
func TestAdvanceCommand_MissingDesignDocRefused(t *testing.T) {
	dir := prepWorkspace(t)
	seedChange(t, dir, "feature-x", "open")
	commitAll(t, dir, "seed change")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"advance", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "feature-x", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading onto-state.yaml: %v", err)
	}
	if st.Phase != "open" {
		t.Errorf("st.Phase = %q, want unchanged %q", st.Phase, "open")
	}
}

// TestAdvanceCommand_BuildToVerifyBlockedByUncheckedTask verifies that a
// build change whose tasks.md still has an unchecked item cannot advance to
// verify, even though every required artifact for verify is present.
func TestAdvanceCommand_BuildToVerifyBlockedByUncheckedTask(t *testing.T) {
	dir := prepWorkspace(t)
	seedChange(t, dir, "feature-x", "build")
	changeDir := filepath.Join(dir, "docs", "changes", "feature-x")
	// verify's RequiredArtifacts adds verification.md on top of build's set.
	writeFile(t, filepath.Join(changeDir, "verification.md"), "")
	writeFile(t, filepath.Join(changeDir, "tasks.md"), "- [ ] todo\n")
	commitAll(t, dir, "seed change")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"advance", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	st, err := ontostate.Load(filepath.Join(changeDir, "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading onto-state.yaml: %v", err)
	}
	if st.Phase != "build" {
		t.Errorf("st.Phase = %q, want unchanged %q", st.Phase, "build")
	}
}

// TestAdvanceCommand_TerminalPhaseRefused verifies that a change already at
// "close" (terminal) cannot advance further and the state is untouched.
func TestAdvanceCommand_TerminalPhaseRefused(t *testing.T) {
	dir := prepWorkspace(t)
	seedChange(t, dir, "feature-x", "close")
	commitAll(t, dir, "seed change")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"advance", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "feature-x", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading onto-state.yaml: %v", err)
	}
	if st.Phase != "close" {
		t.Errorf("st.Phase = %q, want unchanged %q", st.Phase, "close")
	}
}

// TestAdvanceCommand_VerifyToCloseBlockedByDirtyWorktree verifies that
// entering "close" is refused when the worktree is dirty, even though every
// required artifact is present, and leaves the phase unchanged.
func TestAdvanceCommand_VerifyToCloseBlockedByDirtyWorktree(t *testing.T) {
	dir := prepWorkspace(t)
	seedChange(t, dir, "feature-x", "verify")
	commitAll(t, dir, "seed change")
	dirtyWorktree(t, dir)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"advance", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "feature-x", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading onto-state.yaml: %v", err)
	}
	if st.Phase != "verify" {
		t.Errorf("st.Phase = %q, want unchanged %q", st.Phase, "verify")
	}
}

// TestAdvanceCommand_DirtyWorktreeWarnsAndProceedsForNonCloseTransition
// verifies that a dirty worktree only blocks entering "close"; any other
// transition proceeds, printing a warning to stderr.
func TestAdvanceCommand_DirtyWorktreeWarnsAndProceedsForNonCloseTransition(t *testing.T) {
	dir := prepWorkspace(t)
	seedChange(t, dir, "feature-x", "open", "design.md")
	commitAll(t, dir, "seed change")
	dirtyWorktree(t, dir)

	cmd := NewRootCmd()
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"advance", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "feature-x", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading onto-state.yaml: %v", err)
	}
	if st.Phase != "design" {
		t.Errorf("st.Phase = %q, want %q", st.Phase, "design")
	}
	if !bytes.Contains(errOut.Bytes(), []byte("warning")) {
		t.Errorf("stderr = %q, want it to contain %q", errOut.String(), "warning")
	}
}
