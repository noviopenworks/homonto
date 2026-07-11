package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// twoAgentTOML declares two copy-mode local agents, each targeting claude only.
const twoAgentTOML = `
[agents.a]
source = "local:a"
mode = "copy"
targets = ["claude"]

[agents.b]
source = "local:b"
mode = "copy"
targets = ["claude"]
`

// TestAgentsUpdateAllMergesEveryInstalled: `agents update --all` runs the merge
// over every installed agent — an agent whose source changed (disjoint from any
// local edit) is refreshed/merged, an untouched agent is reported up to date, a
// summary line is printed, and the command exits 0.
func TestAgentsUpdateAllMergesEveryInstalled(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, twoAgentTOML, map[string]string{
		"a": "# agent a v1\n",
		"b": "# agent b v1\n",
	})

	for _, name := range []string{"a", "b"} {
		if out, err := runCmd(t, home, "", "agents", "add", name, "--config", cfg); err != nil {
			t.Fatalf("agents add %s: %v\n%s", name, err, out)
		}
	}

	// Change agent a's source; leave b (and both installs) untouched.
	newA := "# agent a v2 (new source)\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "homonto", "agents", "a.md"), []byte(newA), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "--all", "--config", cfg)
	if err != nil {
		t.Fatalf("update --all must exit 0, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "up to date") {
		t.Fatalf("b (untouched) must be reported up to date, got:\n%s", out)
	}
	if !strings.Contains(out, "processed") {
		t.Fatalf("a summary line must be printed, got:\n%s", out)
	}

	aDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "a.md")
	if got, _ := os.ReadFile(aDst); string(got) != newA {
		t.Fatalf("a's dst must reflect the new source, got:\n%s", got)
	}
}

// TestAgentsUpdateAllConflictExitsNonZeroOthersProcessed: with one conflicting
// agent and one cleanly-mergeable agent, `agents update --all` writes a .merged
// sidecar for the conflict and exits non-zero, but STILL processes the other.
func TestAgentsUpdateAllConflictExitsNonZeroOthersProcessed(t *testing.T) {
	home := t.TempDir()
	cfg, cfgDir := addWorkspace(t, twoAgentTOML, map[string]string{
		"a": "line1\nline2\nline3\nline4\nline5\nline6\n",
		"b": "line1\nline2\nline3\nline4\nline5\nline6\n",
	})

	for _, name := range []string{"a", "b"} {
		if out, err := runCmd(t, home, "", "agents", "add", name, "--config", cfg); err != nil {
			t.Fatalf("agents add %s: %v\n%s", name, err, out)
		}
	}

	aDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "a.md")
	bDst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "b.md")

	// a: overlapping edit on line3 (local vs source) → CONFLICT.
	if err := os.WriteFile(aDst, []byte("line1\nline2\nLOCAL3\nline4\nline5\nline6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "homonto", "agents", "a.md"),
		[]byte("line1\nline2\nUPSTREAM3\nline4\nline5\nline6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// b: disjoint local (line1) + source (line6) edits → clean auto-merge.
	if err := os.WriteFile(bDst, []byte("LOCAL1\nline2\nline3\nline4\nline5\nline6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "homonto", "agents", "b.md"),
		[]byte("line1\nline2\nline3\nline4\nline5\nUPSTREAM6\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "--all", "--config", cfg)
	if err == nil {
		t.Fatalf("update --all with a conflict must exit non-zero, got:\n%s", out)
	}

	// a conflicted: sidecar exists, live dst unchanged.
	if _, merr := os.Stat(aDst + ".merged"); merr != nil {
		t.Fatalf("conflicting agent a must get %s.merged: %v", aDst, merr)
	}
	// b still processed: dst merged with BOTH edits.
	gotB, _ := os.ReadFile(bDst)
	if !strings.Contains(string(gotB), "LOCAL1") || !strings.Contains(string(gotB), "UPSTREAM6") {
		t.Fatalf("b must still be processed (merged) despite a's conflict, got:\n%s", gotB)
	}
}

// TestAgentsUpdateAllSkipsOrphan: an installed agent no longer declared in the
// config is skipped with a note, and (absent other issues) the command exits 0.
func TestAgentsUpdateAllSkipsOrphan(t *testing.T) {
	home := t.TempDir()
	toml := "[agents.x]\nsource=\"local:x\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"
	cfg, _ := addWorkspace(t, toml, map[string]string{"x": "# x\n"})

	if out, err := runCmd(t, home, "", "agents", "add", "x", "--config", cfg); err != nil {
		t.Fatalf("agents add x: %v\n%s", err, out)
	}

	// Rewrite the config in place so x is no longer declared (same dir → same lock).
	if err := os.WriteFile(cfg, []byte("[agents.y]\nsource=\"local:y\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, home, "", "agents", "update", "--all", "--config", cfg)
	if err != nil {
		t.Fatalf("update --all with only an orphan must exit 0, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "x: skipped (no longer declared)") {
		t.Fatalf("orphan x must be reported skipped, got:\n%s", out)
	}
}

// TestAgentsUpdateAllUsageErrors: a name together with --all is a usage error,
// and neither a name nor --all is a usage error.
func TestAgentsUpdateAllUsageErrors(t *testing.T) {
	home := t.TempDir()
	cfg, _ := addWorkspace(t, copyAgentTOML, map[string]string{"rev": "# rev\n"})

	if out, err := runCmd(t, home, "", "agents", "update", "foo", "--all", "--config", cfg); err == nil {
		t.Fatalf("name + --all must be a usage error, got:\n%s", out)
	}
	if out, err := runCmd(t, home, "", "agents", "update", "--config", cfg); err == nil {
		t.Fatalf("neither name nor --all must be a usage error, got:\n%s", out)
	}
}

// TestAgentsUpdateSingleStillWorks: a single `agents update <name>` (no --all)
// still merges disjoint edits exactly as before — smoke coverage on top of the
// full existing single-update suite (the refactor guard).
func TestAgentsUpdateSingleStillWorks(t *testing.T) {
	home := t.TempDir()
	base := "line1\nline2\nline3\nline4\nline5\nline6\n"
	toml := "[agents.rev]\nsource=\"local:rev\"\nmode=\"copy\"\ntargets=[\"claude\"]\n"
	cfg, cfgDir := addWorkspace(t, toml, map[string]string{"rev": base})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	dst := filepath.Join(subagentpath.Dir("claude", "user", home, ""), "rev.md")
	if err := os.WriteFile(dst, []byte("LOCAL1\nline2\nline3\nline4\nline5\nline6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "homonto", "agents", "rev.md"),
		[]byte("line1\nline2\nline3\nline4\nline5\nUPSTREAM6\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg); err != nil {
		t.Fatalf("single update must succeed, got %v\n%s", err, out)
	}
	got, _ := os.ReadFile(dst)
	if !strings.Contains(string(got), "LOCAL1") || !strings.Contains(string(got), "UPSTREAM6") {
		t.Fatalf("single update must merge both edits, got:\n%s", got)
	}
}
