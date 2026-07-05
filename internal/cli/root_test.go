package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := out.String(); got != "homonto "+Version+"\n" {
		t.Fatalf("got %q", got)
	}
}

// TestApplyFailsWhenAdapterSkipped: apply proceeds past a skipped adapter
// (the other tools still get their config), but automation must see a
// non-zero exit — "Applied." with exit 0 would hide that one tool was never
// written. Plan/status keep exit 0 with warnings.
func TestApplyFailsWhenAdapterSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Corrupt .claude.json: the claude adapter's Plan fails and is skipped.
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`[not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	cfg := filepath.Join(repo, "homonto.toml")
	if err := os.WriteFile(cfg, []byte("[settings.opencode]\nmodel=\"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"apply", "--yes", "--config", cfg})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("apply with a skipped adapter must exit non-zero; output:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "completed with skipped adapters") {
		t.Fatalf("error = %v; want a skipped-adapters summary", err)
	}
	// The healthy adapter must still have been applied.
	if _, statErr := os.Stat(filepath.Join(home, ".config", "opencode", "opencode.jsonc")); statErr != nil {
		t.Fatalf("opencode config not written despite proceed-past-skip: %v", statErr)
	}
}
