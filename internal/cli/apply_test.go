package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCmd executes a root subcommand with a fresh $HOME, feeding stdinLines to
// any confirmation prompt. It returns combined stdout+stderr and the error.
func runCmd(t *testing.T, home, stdin string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("HOME", home)
	if stdin != "" {
		f, err := os.CreateTemp(t.TempDir(), "stdin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(stdin); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		orig := os.Stdin
		os.Stdin = f
		defer func() { os.Stdin = orig; f.Close() }()
	}
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// seedAdoptable projects cfg to disk and then removes state.json, leaving the
// declared keys on disk == desired but unrecorded — exactly the adoption
// precondition. Returns the config path so the caller can run plan/apply.
func seedAdoptable(t *testing.T, home, config string) string {
	t.Helper()
	repo := t.TempDir()
	cfg := filepath.Join(repo, "homonto.toml")
	if err := os.WriteFile(cfg, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg); err != nil {
		t.Fatalf("seed apply: %v\n%s", err, out)
	}
	// Drop the recorded state: disk still matches desired, but nothing is
	// recorded, so the next plan must emit adopt for each declared key.
	if err := os.Remove(filepath.Join(repo, ".homonto", "state.json")); err != nil {
		t.Fatalf("remove state: %v", err)
	}
	return cfg
}

// Adoption-only apply must reconcile silently: no diff, no prompt (even without
// --yes), a recorded state, and a "Reconciled" summary.
func TestApplyAdoptionOnlyReconcilesWithoutPrompt(t *testing.T) {
	home := t.TempDir()
	cfg := seedAdoptable(t, home, "[settings.opencode]\ntheme=\"dark\"\n")

	// No --yes and no stdin: a prompt here would abort (empty read), proving a
	// regression. The adoption path must not prompt at all.
	out, err := runCmd(t, home, "", "apply", "--config", cfg)
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	if strings.Contains(out, "Apply these changes?") {
		t.Fatalf("adoption-only apply must not prompt:\n%s", out)
	}
	if strings.Contains(out, "~") || strings.Contains(out, "+ setting") {
		t.Fatalf("adoption-only apply must render no diff:\n%s", out)
	}
	if !strings.Contains(out, "Reconciled 1 pre-existing resource(s) into state.") {
		t.Fatalf("want reconcile summary, got:\n%s", out)
	}
	// State must now record the adopted key.
	stateFile := filepath.Join(filepath.Dir(cfg), ".homonto", "state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("state not written: %v", err)
	}
	if !strings.Contains(string(data), "setting.theme") {
		t.Fatalf("adoption did not record setting.theme:\n%s", data)
	}
}

// Adoption-only plan is a no-op view: "No changes." and no diff.
func TestPlanAdoptionOnlyShowsNoChanges(t *testing.T) {
	home := t.TempDir()
	cfg := seedAdoptable(t, home, "[settings.opencode]\ntheme=\"dark\"\n")

	out, err := runCmd(t, home, "", "plan", "--config", cfg)
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No changes. Everything up to date.") {
		t.Fatalf("adoption-only plan must report no changes, got:\n%s", out)
	}
	if strings.Contains(out, "setting.theme") {
		t.Fatalf("adoption-only plan must render no diff, got:\n%s", out)
	}
}

// A mixed run (a real create plus an adoption) renders the diff, prompts
// without --yes, and applies the adoption silently alongside the create.
func TestApplyMixedRendersDiffPromptsAndAdoptsAlongside(t *testing.T) {
	home := t.TempDir()
	// Seed theme, then declare theme (adopt) plus model (create, not on disk).
	cfg := seedAdoptable(t, home, "[settings.opencode]\ntheme=\"dark\"\n")
	if err := os.WriteFile(cfg, []byte("[settings.opencode]\ntheme=\"dark\"\nmodel=\"opus\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "y\n", "apply", "--config", cfg)
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	if !strings.Contains(out, "+ setting.model") {
		t.Fatalf("mixed apply must render the create diff, got:\n%s", out)
	}
	if !strings.Contains(out, "Apply these changes?") {
		t.Fatalf("mixed apply must prompt without --yes, got:\n%s", out)
	}
	if !strings.Contains(out, "Applied.") {
		t.Fatalf("mixed apply must confirm Applied., got:\n%s", out)
	}
	// The visible-change path must not print the reconcile summary.
	if strings.Contains(out, "Reconciled") {
		t.Fatalf("mixed apply must fold adoption in silently, not print Reconciled:\n%s", out)
	}
	// Both keys must be recorded: the created one and the silently adopted one.
	data, err := os.ReadFile(filepath.Join(filepath.Dir(cfg), ".homonto", "state.json"))
	if err != nil {
		t.Fatalf("state not written: %v", err)
	}
	if !strings.Contains(string(data), "setting.model") || !strings.Contains(string(data), "setting.theme") {
		t.Fatalf("both created and adopted keys must be recorded:\n%s", data)
	}
}
