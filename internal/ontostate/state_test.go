package ontostate

import (
	"os"
	"path/filepath"
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
