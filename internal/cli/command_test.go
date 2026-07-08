package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitScaffoldsRepoAndSkipsExisting: `init <dir>` creates the starter files
// and reports each, and a second run is a no-op (existing files are skipped),
// so re-running init never clobbers a user's edits.
func TestInitScaffoldsRepoAndSkipsExisting(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()

	out, err := runCmd(t, home, "", "init", dir)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	for _, want := range []string{"homonto.toml", ".gitignore", ".env.example",
		filepath.Join("content", "skills", ".gitkeep")} {
		p := filepath.Join(dir, want)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("init did not create %s: %v", want, err)
		}
		if !strings.Contains(out, p) {
			t.Fatalf("init output did not report %s\n%s", p, out)
		}
	}

	// Mark the config so we can prove the second run leaves it untouched.
	cfg := filepath.Join(dir, "homonto.toml")
	if err := os.WriteFile(cfg, []byte("# user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out2, err := runCmd(t, home, "", "init", dir)
	if err != nil {
		t.Fatalf("second init: %v\n%s", err, out2)
	}
	if strings.Contains(out2, cfg) {
		t.Fatalf("second init re-created an existing file:\n%s", out2)
	}
	if b, _ := os.ReadFile(cfg); string(b) != "# user edit\n" {
		t.Fatalf("second init clobbered an existing config: %q", string(b))
	}
}

// seedClaudeMCP writes a minimal ~/.claude.json with one stdio MCP server, the
// shape `import` reads (command string + args array).
func seedClaudeMCP(t *testing.T, home string) {
	t.Helper()
	doc := `{"mcpServers":{"codegraph":{"type":"stdio","command":"codegraph","args":["serve","--mcp"]}}}`
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestImportWritesConfigThenRefusesOverwriteThenForces exercises the full
// import command surface: it writes a homonto.toml from the current setup, then
// refuses to overwrite an existing one without --force, then overwrites with it.
func TestImportWritesConfigThenRefusesOverwriteThenForces(t *testing.T) {
	home := t.TempDir()
	seedClaudeMCP(t, home)
	cfg := filepath.Join(t.TempDir(), "homonto.toml")

	// 1. Fresh import writes the file and mentions the imported MCP.
	out, err := runCmd(t, home, "", "import", "--config", cfg)
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	body, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("import did not write config: %v", err)
	}
	if !strings.Contains(string(body), "codegraph") {
		t.Fatalf("imported config missing the MCP server:\n%s", body)
	}

	// 2. Second import without --force must NOT overwrite; it says so and leaves
	//    a sentinel edit intact.
	if err := os.WriteFile(cfg, []byte("# hand edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = runCmd(t, home, "", "import", "--config", cfg)
	if err != nil {
		t.Fatalf("import (no force): %v\n%s", err, out)
	}
	if !strings.Contains(out, "already exists") || !strings.Contains(out, "--force") {
		t.Fatalf("import without --force should refuse and mention --force:\n%s", out)
	}
	if b, _ := os.ReadFile(cfg); string(b) != "# hand edited\n" {
		t.Fatalf("import without --force overwrote the file: %q", string(b))
	}

	// 3. import --force overwrites, restoring the imported content.
	out, err = runCmd(t, home, "", "import", "--force", "--config", cfg)
	if err != nil {
		t.Fatalf("import --force: %v\n%s", err, out)
	}
	if b, _ := os.ReadFile(cfg); !strings.Contains(string(b), "codegraph") {
		t.Fatalf("import --force did not overwrite with imported content: %q", string(b))
	}
}

// TestPlanReportsConfigLoadError: a command over an invalid config surfaces a
// clear, non-nil error naming the problem instead of proceeding or panicking —
// the command-level guard on bad input reaching the CLI.
func TestPlanReportsConfigLoadError(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "homonto.toml")
	// enabledPlugins is a reserved settings.claude key (config.Load rejects it).
	if err := os.WriteFile(cfg, []byte("[settings.claude]\nenabledPlugins = {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runCmd(t, home, "", "plan", "--config", cfg)
	if err == nil {
		t.Fatalf("plan accepted an invalid config; want error\n%s", out)
	}
	if !strings.Contains(err.Error(), "enabledPlugins") {
		t.Fatalf("error does not name the offending key: %v", err)
	}
}

// TestPlanReportsMissingConfig: a missing config file is a clear error, not a
// silent empty plan.
func TestPlanReportsMissingConfig(t *testing.T) {
	home := t.TempDir()
	missing := filepath.Join(t.TempDir(), "does-not-exist.toml")
	out, err := runCmd(t, home, "", "plan", "--config", missing)
	if err == nil {
		t.Fatalf("plan accepted a missing config; want error\n%s", out)
	}
}
