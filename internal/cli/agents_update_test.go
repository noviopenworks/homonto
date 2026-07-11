package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// TestAgentsUpdateSourceChangedInstallUntouched: when the provider source changes
// but the installed copies were not touched, update re-materializes the new
// content to every target, refreshes the lockfile hash, and creates NO .bak
// (an untouched install of an older source is not a local edit).
func TestAgentsUpdateSourceChangedInstallUntouched(t *testing.T) {
	home := t.TempDir()
	old := "# Rev agent v1\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": old})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Change the provider source; leave installed copies untouched.
	neu := "# Rev agent v2 (new source)\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(neu), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}

	wantHash := agentlock.HashContent([]byte(neu))
	for _, tool := range []string{"claude", "opencode"} {
		dst := filepath.Join(subagentpath.Dir(tool, "user", home, ""), "rev.md")
		got, rerr := os.ReadFile(dst)
		if rerr != nil {
			t.Fatalf("%s: expected file at %s: %v", tool, dst, rerr)
		}
		if string(got) != neu {
			t.Fatalf("%s: expected refreshed content, got:\n%s", tool, got)
		}
		if _, err := os.Stat(dst + ".bak"); !os.IsNotExist(err) {
			t.Fatalf("%s: untouched install must NOT be backed up, but %s.bak exists", tool, dst)
		}
	}

	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"claude", "opencode"} {
		if ins := lock.Agents["rev"].Installed[tool]; ins.Hash != wantHash {
			t.Fatalf("%s: lockfile hash not refreshed: got %s want %s", tool, ins.Hash, wantHash)
		}
	}
}

// TestAgentsUpdateBacksUpLocalEdit: when a target copy was locally edited AND the
// source also changed, update writes the new source to the target and preserves
// the local edit in <dst>.bak.
func TestAgentsUpdateBacksUpLocalEdit(t *testing.T) {
	home := t.TempDir()
	old := "# Rev agent v1\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": old})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	// Locally edit the claude copy (X), differing from the recorded install.
	localX := "# LOCAL edit by user\n"
	claudeDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	if err := os.WriteFile(claudeDst, []byte(localX), 0o644); err != nil {
		t.Fatal(err)
	}
	// Change the source to Y.
	srcY := "# Rev agent v2\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(srcY), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}

	if got, _ := os.ReadFile(claudeDst); string(got) != srcY {
		t.Fatalf("claude dst must hold new source Y, got:\n%s", got)
	}
	bak, berr := os.ReadFile(claudeDst + ".bak")
	if berr != nil {
		t.Fatalf("local edit must be backed up to %s.bak: %v", claudeDst, berr)
	}
	if string(bak) != localX {
		t.Fatalf("%s.bak must preserve the local edit X, got:\n%s", claudeDst, bak)
	}
}

// TestAgentsUpdateIsIdempotent: updating immediately after add reports every
// target "up to date", writes no .bak, and leaves dst bytes unchanged.
func TestAgentsUpdateIsIdempotent(t *testing.T) {
	home := t.TempDir()
	body := "# Rev agent\n"
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": body})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	claudeDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	fiBefore, err := os.Stat(claudeDst)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Fatalf("idempotent update must report up to date, got:\n%s", out)
	}
	if _, err := os.Stat(claudeDst + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("idempotent update must NOT create %s.bak", claudeDst)
	}
	fiAfter, err := os.Stat(claudeDst)
	if err != nil {
		t.Fatal(err)
	}
	if !fiAfter.ModTime().Equal(fiBefore.ModTime()) {
		t.Fatalf("idempotent update must not rewrite dst (mtime changed): %v -> %v", fiBefore.ModTime(), fiAfter.ModTime())
	}
	if got, _ := os.ReadFile(claudeDst); string(got) != body {
		t.Fatalf("content changed on idempotent update:\n%s", got)
	}
}

// TestAgentsUpdateNotInstalled: updating a declared-but-not-installed agent errors,
// pointing at `agents add`.
func TestAgentsUpdateNotInstalled(t *testing.T) {
	home := t.TempDir()
	toml := "[agents.new]\nsource=\"local:new\"\nmode=\"copy\"\n"
	cfg, _ := addWorkspace(t, toml, map[string]string{"new": "# new\n"})

	out, err := runCmd(t, home, "", "agents", "update", "new", "--config", cfg)
	if err == nil {
		t.Fatalf("update of a not-installed agent must error, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "not installed") || !strings.Contains(err.Error(), "agents add") {
		t.Fatalf("error must say not installed and point at `agents add`, got: %v", err)
	}
}

// TestAgentsUpdateBuiltinNotSupported: a builtin: source is refused.
func TestAgentsUpdateBuiltinNotSupported(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, "[agents.b]\nsource=\"builtin:b\"\n", nil)
	out, err := runCmd(t, home, "", "agents", "update", "b", "--config", cfg)
	if err == nil {
		t.Fatalf("builtin source must be refused, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "local:") {
		t.Fatalf("error must explain only local: sources are supported, got: %v", err)
	}
}

// TestAgentsUpdateUndeclared: updating an undeclared agent errors clearly.
func TestAgentsUpdateUndeclared(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "x"})
	out, err := runCmd(t, home, "", "agents", "update", "nope", "--config", cfg)
	if err == nil {
		t.Fatalf("undeclared agent must error, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("error must say not declared, got: %v", err)
	}
}

// TestAgentsUpdateLinkModeUpToDate: a link-mode agent whose symlink already points
// at the source is reported up to date and stays a symlink to the source.
func TestAgentsUpdateLinkModeUpToDate(t *testing.T) {
	home := t.TempDir()
	toml := `
[agents.rev]
source = "local:rev"
mode = "link"
targets = ["claude"]
`
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Fatalf("link update must report up to date, got:\n%s", out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("link dst missing: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link mode dst must remain a symlink at %s", dst)
	}
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if target != srcPath {
		t.Fatalf("symlink must point at source %s, got %s", srcPath, target)
	}
}
