package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// codeReviewerSnippet is a stable phrase from the embedded catalog's
// code-reviewer agent, used to assert builtin content was installed.
const codeReviewerSnippet = "You are a focused code reviewer"

// TestAgentsAddBuiltinInstallsCatalogContent: a copy-mode builtin agent resolves
// its content from the embedded catalog (no local homonto/agents file), installs
// it into each target, and records the lockfile.
func TestAgentsAddBuiltinInstallsCatalogContent(t *testing.T) {
	home := t.TempDir()
	toml := `
[agents.cr]
source = "builtin:code-reviewer"
mode = "copy"
targets = ["claude"]
`
	cfg, cfgDir := addWorkspace(t, toml, nil)

	out, err := runCmd(t, home, "", "agents", "add", "cr", "--config", cfg)
	if err != nil {
		t.Fatalf("agents add builtin: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "cr.md")
	got, rerr := os.ReadFile(dst)
	if rerr != nil {
		t.Fatalf("expected installed file at %s: %v", dst, rerr)
	}
	if !strings.Contains(string(got), codeReviewerSnippet) {
		t.Fatalf("installed builtin content must be the catalog's code-reviewer, got:\n%s", got)
	}

	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatalf("load lock: %v", err)
	}
	rec, ok := lock.Agents["cr"]
	if !ok {
		t.Fatalf("lockfile must record agent cr, got %+v", lock.Agents)
	}
	if rec.Source != "builtin:code-reviewer" || rec.Mode != "copy" {
		t.Fatalf("lock record source/mode wrong: %+v", rec)
	}
	if _, ok := rec.Installed["claude"]; !ok {
		t.Fatalf("lock must record claude install: %+v", rec.Installed)
	}
}

// TestAgentsAddBuiltinDefaultsToCopy: a builtin agent with NO explicit mode must
// default to copy (link is impossible for builtin), not hit the builtin+link
// error, and must install the catalog content and record mode "copy".
func TestAgentsAddBuiltinDefaultsToCopy(t *testing.T) {
	home := t.TempDir()
	toml := "[agents.cr]\nsource = \"builtin:code-reviewer\"\ntargets = [\"claude\"]\n" // no mode
	cfg, cfgDir := addWorkspace(t, toml, nil)

	out, err := runCmd(t, home, "", "agents", "add", "cr", "--config", cfg)
	if err != nil {
		t.Fatalf("builtin agent with no mode must install (copy default): %v\n%s", err, out)
	}
	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "cr.md")
	if got, rerr := os.ReadFile(dst); rerr != nil || !strings.Contains(string(got), codeReviewerSnippet) {
		t.Fatalf("builtin content not installed: err=%v content=%q", rerr, got)
	}
	lock, _ := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if lock.Agents["cr"].Mode != "copy" {
		t.Fatalf("builtin with no mode must record mode=copy, got %q", lock.Agents["cr"].Mode)
	}
}

// TestAgentsAddUnknownBuiltinIsError: an unknown builtin agent name errors
// clearly.
func TestAgentsAddUnknownBuiltinIsError(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, "[agents.x]\nsource=\"builtin:not-a-real-agent\"\nmode=\"copy\"\n", nil)
	out, err := runCmd(t, home, "", "agents", "add", "x", "--config", cfg)
	if err == nil {
		t.Fatalf("unknown builtin must error, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "unknown builtin agent") {
		t.Fatalf("error must say unknown builtin agent, got: %v", err)
	}
}

// TestAgentsAddBuiltinLinkIsError: builtin sources have no local path to symlink,
// so builtin + link mode is rejected up front.
func TestAgentsAddBuiltinLinkIsError(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, "[agents.cr]\nsource=\"builtin:code-reviewer\"\nmode=\"link\"\ntargets=[\"claude\"]\n", nil)
	out, err := runCmd(t, home, "", "agents", "add", "cr", "--config", cfg)
	if err == nil {
		t.Fatalf("builtin + link must error, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "link") || !strings.Contains(err.Error(), "builtin") {
		t.Fatalf("error must explain builtin cannot use link mode, got: %v", err)
	}
}

// TestResolveAgentSourceRejectsRemote: a non-local, non-builtin source returns a
// clear "not yet supported" error (defensive; config load also rejects it).
func TestResolveAgentSourceRejectsRemote(t *testing.T) {
	_, err := resolveAgentSource(config.Agent{Source: "https://example.com/x"}, t.TempDir())
	if err == nil {
		t.Fatalf("remote source must be unsupported")
	}
	if !strings.Contains(err.Error(), "not yet supported") {
		t.Fatalf("error must say not yet supported, got: %v", err)
	}
}

// TestAgentsDoctorBuiltinHealthy: a freshly-added builtin agent whose recorded
// base still matches the catalog content is healthy.
func TestAgentsDoctorBuiltinHealthy(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, "[agents.cr]\nsource=\"builtin:code-reviewer\"\nmode=\"copy\"\ntargets=[\"claude\"]\n", nil)

	if out, err := runCmd(t, home, "", "agents", "add", "cr", "--config", cfg); err != nil {
		t.Fatalf("agents add builtin: %v\n%s", err, out)
	}

	out, err := runCmd(t, home, "", "agents", "doctor", "--config", cfg)
	if err != nil {
		t.Fatalf("healthy builtin doctor must succeed, got err %v\n%s", err, out)
	}
	if out != "healthy\n" {
		t.Fatalf("healthy builtin doctor must print exactly \"healthy\", got:\n%q", out)
	}
}

// TestAgentsUpdateBuiltinIsIdempotent: updating a freshly-added, unchanged
// builtin agent is a clean no-op ("up to date").
func TestAgentsUpdateBuiltinIsIdempotent(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, "[agents.cr]\nsource=\"builtin:code-reviewer\"\nmode=\"copy\"\ntargets=[\"claude\"]\n", nil)

	if out, err := runCmd(t, home, "", "agents", "add", "cr", "--config", cfg); err != nil {
		t.Fatalf("agents add builtin: %v\n%s", err, out)
	}

	out, err := runCmd(t, home, "", "agents", "update", "cr", "--config", cfg)
	if err != nil {
		t.Fatalf("update of unchanged builtin must succeed, got err %v\n%s", err, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Fatalf("unchanged builtin update must report up to date, got:\n%s", out)
	}
}
