package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// addWorkspace builds a temp config dir with homonto.toml (body) plus any
// agent source files, and returns the config path and its dir.
func addWorkspace(t *testing.T, toml string, sources map[string]string) (cfg, cfgDir string) {
	t.Helper()
	cfgDir = t.TempDir()
	cfg = filepath.Join(cfgDir, "homonto.toml")
	if err := os.WriteFile(cfg, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	agentsDir := filepath.Join(cfgDir, "homonto", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range sources {
		if err := os.WriteFile(filepath.Join(agentsDir, name+".md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return cfg, cfgDir
}

const copyAgentTOML = `
[agents.rev]
source = "local:rev"
mode = "copy"
targets = ["claude", "opencode"]
`

// TestAgentsAddCopyInstallsAndRecords: a copy-mode local agent is written to
// every target's user agent dir with the source content, and the lockfile
// records each target's path + content hash.
func TestAgentsAddCopyInstallsAndRecords(t *testing.T) {
	home := t.TempDir()
	body := "# Rev agent\nreview carefully\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": body})

	out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	wantHash := agentlock.HashContent([]byte(body))
	for _, tool := range []string{"claude", "opencode"} {
		dst := filepath.Join(subagentpath.Dir(tool, "user", home, ""), "rev.md")
		got, rerr := os.ReadFile(dst)
		if rerr != nil {
			t.Fatalf("%s: expected installed file at %s: %v", tool, dst, rerr)
		}
		if string(got) != body {
			t.Fatalf("%s: installed content mismatch at %s:\n%s", tool, dst, got)
		}
	}

	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatalf("load lock: %v", err)
	}
	rec, ok := lock.Agents["rev"]
	if !ok {
		t.Fatalf("lockfile must record agent rev, got %+v", lock.Agents)
	}
	if rec.Source != "local:rev" || rec.Mode != "copy" {
		t.Fatalf("lock record source/mode wrong: %+v", rec)
	}
	for _, tool := range []string{"claude", "opencode"} {
		ins, ok := rec.Installed[tool]
		if !ok {
			t.Fatalf("lock must record install for %s: %+v", tool, rec.Installed)
		}
		wantPath := filepath.Join(subagentpath.Dir(tool, "user", home, ""), "rev.md")
		if ins.Path != wantPath || ins.Hash != wantHash {
			t.Fatalf("%s install record wrong: got %+v want path=%s hash=%s", tool, ins, wantPath, wantHash)
		}
	}
}

// TestAgentsAddIsIdempotent: a second add of an up-to-date agent reports "up to
// date", does not rewrite the dst files, and leaves a valid lockfile.
func TestAgentsAddIsIdempotent(t *testing.T) {
	home := t.TempDir()
	body := "# Rev agent\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": body})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("first add: %v\n%s", err, out)
	}

	claudeDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	fiBefore, err := os.Stat(claudeDst)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("second add: %v\n%s", err, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Fatalf("re-run must report up to date, got:\n%s", out)
	}

	fiAfter, err := os.Stat(claudeDst)
	if err != nil {
		t.Fatal(err)
	}
	if !fiAfter.ModTime().Equal(fiBefore.ModTime()) {
		t.Fatalf("idempotent re-run must not rewrite dst (mtime changed): %v -> %v", fiBefore.ModTime(), fiAfter.ModTime())
	}
	got, _ := os.ReadFile(claudeDst)
	if string(got) != body {
		t.Fatalf("content changed on re-run:\n%s", got)
	}

	if _, err := agentlock.Load(filepath.Join(cfgDir, ".homonto")); err != nil {
		t.Fatalf("lockfile invalid after re-run: %v", err)
	}
}

// TestAgentsAddConflictIsAllOrNothing: an unmanaged file at one target dst makes
// add refuse with an error naming the conflict, installs NOTHING (the other
// target gets no file), and does not create/change the lockfile.
func TestAgentsAddConflictIsAllOrNothing(t *testing.T) {
	home := t.TempDir()
	body := "# Rev agent\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": body})

	// Pre-create an unmanaged file at the claude target (not via add).
	claudeDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	if err := os.MkdirAll(filepath.Dir(claudeDst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claudeDst, []byte("USER OWNED\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg)
	if err == nil {
		t.Fatalf("add must fail on an unmanaged-file conflict, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), claudeDst) {
		t.Fatalf("conflict error must name the conflicting path %s, got: %v", claudeDst, err)
	}

	// The pre-existing file is untouched.
	if got, _ := os.ReadFile(claudeDst); string(got) != "USER OWNED\n" {
		t.Fatalf("conflict must not overwrite the foreign file, got:\n%s", got)
	}
	// All-or-nothing: the other target must NOT have been installed.
	opencodeDst := filepath.Join(subagentpath.Dir("opencode", "user", home, ""), "rev.md")
	if _, err := os.Stat(opencodeDst); !os.IsNotExist(err) {
		t.Fatalf("conflict must install nothing; opencode target should not exist: %v", err)
	}
	// Lockfile must not have been created.
	if _, err := os.Stat(filepath.Join(cfgDir, ".homonto", "agents-lock.json")); !os.IsNotExist(err) {
		t.Fatalf("conflict must not create/change the lockfile: %v", err)
	}
}

// TestAgentsAddBuiltinNotSupported: a builtin: source is refused with a clear
// "not supported yet" error.
func TestAgentsAddBuiltinNotSupported(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, "[agents.b]\nsource=\"builtin:b\"\n", nil)
	out, err := runCmd(t, home, "", "agents", "add", "b", "--config", cfg)
	if err == nil {
		t.Fatalf("builtin source must be refused, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "local:") {
		t.Fatalf("error must explain only local: sources are supported, got: %v", err)
	}
}

// TestAgentsAddUndeclared: adding an agent that is not declared errors clearly.
func TestAgentsAddUndeclared(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "x"})
	out, err := runCmd(t, home, "", "agents", "add", "nope", "--config", cfg)
	if err == nil {
		t.Fatalf("undeclared agent must error, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("error must say not declared, got: %v", err)
	}
}

// TestAgentsAddMissingSourceFile: a declared local agent whose source file is
// absent errors naming the missing path.
func TestAgentsAddMissingSourceFile(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, "[agents.gone]\nsource=\"local:gone\"\nmode=\"copy\"\n", nil)
	out, err := runCmd(t, home, "", "agents", "add", "gone", "--config", cfg)
	if err == nil {
		t.Fatalf("missing source file must error, got:\n%s", out)
	}
	wantPath := filepath.Join(cfgDir, "homonto", "agents", "gone.md")
	if !strings.Contains(err.Error(), wantPath) {
		t.Fatalf("error must name the missing source path %s, got: %v", wantPath, err)
	}
}

// TestAgentsAddLinkModeSymlinks: a link-mode agent installs a symlink pointing
// at the source file and records it in the lockfile.
func TestAgentsAddLinkModeSymlinks(t *testing.T) {
	home := t.TempDir()
	toml := `
[agents.rev]
source = "local:rev"
mode = "link"
targets = ["claude"]
`
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": "# rev\n"})

	out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg)
	if err != nil {
		t.Fatalf("link add: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("link dst missing: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link mode must create a symlink at %s", dst)
	}
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if target != srcPath {
		t.Fatalf("symlink must point at source %s, got %s", srcPath, target)
	}

	lock, err := agentlock.Load(filepath.Join(cfgDir, ".homonto"))
	if err != nil {
		t.Fatal(err)
	}
	if ins, ok := lock.Agents["rev"].Installed["claude"]; !ok || ins.Path != dst {
		t.Fatalf("lock must record link install at %s, got %+v", dst, lock.Agents["rev"])
	}
}
