package ontocli

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

// TestValidChangeName_Valid covers the accepted shape: lowercase
// alphanumeric segments separated by single hyphens.
func TestValidChangeName_Valid(t *testing.T) {
	if err := validChangeName("feature-x"); err != nil {
		t.Errorf("validChangeName(%q) = %v, want nil", "feature-x", err)
	}
}

// TestValidChangeName_Invalid covers every rejected shape: empty, path
// traversal, path separators, uppercase, and a leading hyphen.
func TestValidChangeName_Invalid(t *testing.T) {
	cases := []string{"", "../evil", "Foo", "a/b", "-x"}
	for _, name := range cases {
		if err := validChangeName(name); err == nil {
			t.Errorf("validChangeName(%q) = nil, want error", name)
		}
	}
}

var createdDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// TestNewCommand_CreatesSkeleton verifies that, once the framework gate
// passes and the name is valid, "onto new" scaffolds onto-state.yaml
// (phase open, matching change name, created date) plus empty proposal.md
// and tasks.md, and exits 0.
func TestNewCommand_CreatesSkeleton(t *testing.T) {
	dir := setUpGatedWorkspace(t)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"new", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	changeDir := filepath.Join(dir, "docs", "changes", "feature-x")

	state, err := ontostate.Load(filepath.Join(changeDir, "onto-state.yaml"))
	if err != nil {
		t.Fatalf("loading onto-state.yaml: %v", err)
	}
	if state.Change != "feature-x" {
		t.Errorf("state.Change = %q, want %q", state.Change, "feature-x")
	}
	if state.Phase != "open" {
		t.Errorf("state.Phase = %q, want %q", state.Phase, "open")
	}
	if !createdDatePattern.MatchString(state.Created) {
		t.Errorf("state.Created = %q, want match of %q", state.Created, createdDatePattern.String())
	}

	for _, f := range []string{"proposal.md", "tasks.md"} {
		if _, err := os.Stat(filepath.Join(changeDir, f)); err != nil {
			t.Errorf("expected %s to exist, stat err = %v", f, err)
		}
	}
}

// TestNewCommand_RefusesToClobberExisting verifies that a second "onto new"
// with the same change name fails without touching any pre-existing
// artifact in that change directory.
func TestNewCommand_RefusesToClobberExisting(t *testing.T) {
	dir := setUpGatedWorkspace(t)

	cmd1 := NewRootCmd()
	cmd1.SetOut(&bytes.Buffer{})
	cmd1.SetErr(&bytes.Buffer{})
	cmd1.SetArgs([]string{"new", "feature-x", "--dir", dir})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first execute: %v", err)
	}

	proposalPath := filepath.Join(dir, "docs", "changes", "feature-x", "proposal.md")
	knownContent := "known bytes, do not touch\n"
	writeFile(t, proposalPath, knownContent)

	cmd2 := NewRootCmd()
	cmd2.SetOut(&bytes.Buffer{})
	cmd2.SetErr(&bytes.Buffer{})
	cmd2.SetArgs([]string{"new", "feature-x", "--dir", dir})
	if err := cmd2.Execute(); err == nil {
		t.Fatal("second execute() = nil, want error")
	}

	got, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatalf("reading proposal.md: %v", err)
	}
	if string(got) != knownContent {
		t.Errorf("proposal.md content = %q, want unchanged %q", string(got), knownContent)
	}
}

// TestNewCommand_InvalidNameCreatesNothing verifies that an invalid change
// name fails before any write, leaving docs/changes/<name> absent.
func TestNewCommand_InvalidNameCreatesNothing(t *testing.T) {
	dir := setUpGatedWorkspace(t)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"new", "Bad", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "Bad")); !os.IsNotExist(err) {
		t.Errorf("expected docs/changes/Bad to not exist, stat err = %v", err)
	}
}

// TestNewCommand_GateFailureCreatesNothing verifies that when the framework
// gate fails (no homonto.toml here), "onto new" returns a non-zero exit
// and creates no docs/changes directory at all.
func TestNewCommand_GateFailureCreatesNothing(t *testing.T) {
	dir := t.TempDir()

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"new", "feature-x", "--dir", dir})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error")
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "changes")); !os.IsNotExist(err) {
		t.Errorf("expected docs/changes to not exist, stat err = %v", err)
	}
}
