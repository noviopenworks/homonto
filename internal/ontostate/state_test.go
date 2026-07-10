package ontostate

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParse_ValidYAML_DerivesBuildPhase(t *testing.T) {
	input := []byte(`
change: onto-binary-foundation
workflow: full
phase: build
created: "2026-07-10"
base_ref: main
deps:
  - other-change
`)

	state, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if state.Change != "onto-binary-foundation" {
		t.Errorf("Change = %q, want %q", state.Change, "onto-binary-foundation")
	}
	if state.Phase != "build" {
		t.Errorf("Phase = %q, want %q", state.Phase, "build")
	}

	phase, err := state.DerivePhase()
	if err != nil {
		t.Fatalf("DerivePhase returned unexpected error: %v", err)
	}
	if phase != "build" {
		t.Errorf("DerivePhase() = %q, want %q", phase, "build")
	}
}

func TestParse_MalformedYAML_ErrorMentionsOntoState(t *testing.T) {
	input := []byte("change: [unterminated\n  phase: build")

	_, err := Parse(input)
	if err == nil {
		t.Fatal("Parse returned nil error for malformed YAML, want error")
	}
	if !strings.Contains(err.Error(), "onto-state") {
		t.Errorf("Parse error = %q, want it to contain %q", err.Error(), "onto-state")
	}
}

func TestValidate_UnknownPhase_ReturnsError(t *testing.T) {
	state := State{Change: "onto-binary-foundation", Phase: "bogus"}

	err := state.Validate()
	if err == nil {
		t.Fatal("Validate returned nil error for unknown phase, want error")
	}
}

func TestValidate_EmptyChange_ReturnsError(t *testing.T) {
	state := State{Change: "", Phase: "build"}

	err := state.Validate()
	if err == nil {
		t.Fatal("Validate returned nil error for empty change, want error")
	}
}

func TestLoad_MissingFile_ErrorNamesPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist", "onto-state.yaml")

	_, err := Load(missing)
	if err == nil {
		t.Fatal("Load returned nil error for missing file, want error")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("Load error = %q, want it to contain path %q", err.Error(), missing)
	}
}

func TestLoad_ValidFile_ReturnsState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "onto-state.yaml")
	content := []byte("change: onto-binary-foundation\nphase: open\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	state, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if state.Change != "onto-binary-foundation" {
		t.Errorf("Change = %q, want %q", state.Change, "onto-binary-foundation")
	}
	if state.Phase != "open" {
		t.Errorf("Phase = %q, want %q", state.Phase, "open")
	}
}

func TestLoad_MalformedFile_ErrorNamesPathAndProblem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "onto-state.yaml")
	content := []byte("change: [unterminated\n  phase: build")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load returned nil error for malformed file, want error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("Load error = %q, want it to contain path %q", err.Error(), path)
	}
	if !strings.Contains(err.Error(), "yaml") {
		t.Errorf("Load error = %q, want it to indicate the YAML/parse problem", err.Error())
	}
}

func TestParse_GarbageBytes_DoesNotPanicAndReturnsError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parse panicked on garbage bytes: %v", r)
		}
	}()

	_, err := Parse([]byte("\x00\x01garbage"))
	if err == nil {
		t.Fatal("Parse returned nil error for garbage bytes, want error")
	}
}

func TestDerivePhase_InvalidState_ReturnsValidateError(t *testing.T) {
	state := State{Change: "", Phase: "build"}

	_, err := state.DerivePhase()
	if err == nil {
		t.Fatal("DerivePhase returned nil error for invalid state, want error")
	}
}

func TestMarshalParse_RoundTrip_SimpleState(t *testing.T) {
	state := State{Change: "c", Phase: "build"}

	b, err := Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned unexpected error: %v", err)
	}

	got, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, state) {
		t.Errorf("Parse(Marshal(state)) = %+v, want %+v", got, state)
	}
}

func TestMarshalParse_RoundTrip_FullState(t *testing.T) {
	state := State{
		Change:   "onto-binary-foundation",
		Workflow: "full",
		Phase:    "verify",
		Created:  "2026-07-10",
		BaseRef:  "main",
		Deps:     []string{"other-change", "another-change"},
		Archived: true,
	}

	b, err := Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned unexpected error: %v", err)
	}

	got, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, state) {
		t.Errorf("Parse(Marshal(state)) = %+v, want %+v", got, state)
	}
}

func TestSave_NonExistentSubdir_CreatesDirAndFileAndCleansUpTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "sub", "onto-state.yaml")
	state := State{Change: "onto-binary-foundation", Phase: "open"}

	if err := Save(path, state); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, state) {
		t.Errorf("Load(Save(state)) = %+v, want %+v", got, state)
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("expected temp file %s to be gone, stat err = %v", path+".tmp", err)
	}
}

func TestRequiredArtifacts_OpenPhase_ReturnsBaseSet(t *testing.T) {
	want := []string{"onto-state.yaml", "proposal.md", "tasks.md"}

	got := RequiredArtifacts("open")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RequiredArtifacts(\"open\") = %v, want %v", got, want)
	}
}

func TestValidateSkeleton_AllArtifactsPresent_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	state := State{Change: "onto-binary-foundation", Phase: "open"}
	if err := Save(filepath.Join(dir, "onto-state.yaml"), state); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("proposal"), 0o644); err != nil {
		t.Fatalf("failed to write proposal.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("tasks"), 0o644); err != nil {
		t.Fatalf("failed to write tasks.md: %v", err)
	}

	if err := ValidateSkeleton(dir); err != nil {
		t.Errorf("ValidateSkeleton returned unexpected error: %v", err)
	}
}

func TestValidateSkeleton_MissingTasksFile_ErrorNamesFile(t *testing.T) {
	dir := t.TempDir()
	state := State{Change: "onto-binary-foundation", Phase: "open"}
	if err := Save(filepath.Join(dir, "onto-state.yaml"), state); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("proposal"), 0o644); err != nil {
		t.Fatalf("failed to write proposal.md: %v", err)
	}

	err := ValidateSkeleton(dir)
	if err == nil {
		t.Fatal("ValidateSkeleton returned nil error for missing tasks.md, want error")
	}
	if !strings.Contains(err.Error(), "tasks.md") {
		t.Errorf("ValidateSkeleton error = %q, want it to contain %q", err.Error(), "tasks.md")
	}
}
