package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAgentsListReportsDeclaredAgents: `agents list` prints every declared
// agent sorted by name with its source/version/targets/mode; an agent with no
// version reads as "unpinned".
func TestAgentsListReportsDeclaredAgents(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "homonto.toml")
	doc := `
[agents.review]
source = "builtin:review-agent"
version = "1.2.0"
targets = ["claude", "opencode"]
mode = "copy"

[agents.docs]
source = "local:doc-writer"
`
	if err := os.WriteFile(cfg, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "list", "--config", cfg)
	if err != nil {
		t.Fatalf("agents list: %v\n%s", err, out)
	}

	// Both agents listed, sorted by name (docs before review).
	di := strings.Index(out, "docs:")
	ri := strings.Index(out, "review:")
	if di < 0 || ri < 0 {
		t.Fatalf("both agents must be listed, got:\n%s", out)
	}
	if di > ri {
		t.Fatalf("agents must be sorted by name (docs before review), got:\n%s", out)
	}
	if !strings.Contains(out, "builtin:review-agent") || !strings.Contains(out, "version=1.2.0") ||
		!strings.Contains(out, "targets=claude,opencode") || !strings.Contains(out, "mode=copy") {
		t.Fatalf("pinned agent line missing fields, got:\n%s", out)
	}
	// The version-less agent reads as unpinned and defaults targets+mode.
	if !strings.Contains(out, "local:doc-writer") || !strings.Contains(out, "version=unpinned") ||
		!strings.Contains(out, "mode=link") {
		t.Fatalf("unpinned agent line missing defaults, got:\n%s", out)
	}
}

// TestAgentsListNoAgents: a config declaring no [agents] reports a clear empty
// message rather than nothing.
func TestAgentsListNoAgents(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(cfg, []byte("[settings.opencode]\nmodel=\"opus\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "list", "--config", cfg)
	if err != nil {
		t.Fatalf("agents list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No agents declared.") {
		t.Fatalf("empty config must report No agents declared., got:\n%s", out)
	}
}

// TestAgentsListIsReadOnly: `agents list` loads config only — it must not build
// the engine or write any state/tool files into $HOME.
func TestAgentsListIsReadOnly(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(cfg, []byte("[agents.review]\nsource=\"builtin:x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "list", "--config", cfg)
	if err != nil {
		t.Fatalf("agents list: %v\n%s", err, out)
	}

	// A read-only command creates nothing under $HOME (no state.json, no
	// .claude/.config tool files).
	var created []string
	_ = filepath.Walk(home, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			created = append(created, p)
		}
		return nil
	})
	if len(created) != 0 {
		t.Fatalf("agents list must be read-only but created files under HOME: %v\noutput:\n%s", created, out)
	}
}
