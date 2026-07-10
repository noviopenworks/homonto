package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/secret"
)

const cometTOML = `
[frameworks.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`

const commandTOML = `
[commands.example-command]
source = "builtin:example-command"
scope = "user"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`

const subagentTOML = `
[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`

func buildEngine(t *testing.T, home, repo string) *Engine {
	t.Helper()
	e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
	return e
}

func buildEngineWithSubagent(t *testing.T) *Engine {
	t.Helper()
	home := t.TempDir()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(subagentTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	return buildEngine(t, home, repo)
}

func mustPlan(t *testing.T, e *Engine) []adapter.ChangeSet {
	t.Helper()
	sets, err := e.Plan()
	if err != nil {
		t.Fatal(err)
	}
	return sets
}

func TestApplyMaterializesBuiltinSkills(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(cometTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// A known comet skill materialized under .homonto/catalog/skills/.
	got := filepath.Join(repo, ".homonto", "catalog", "skills", "comet-open", "SKILL.md")
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("comet-open not materialized: %v", err)
	}
	// State recorded the catalog version.
	if e.State.CatalogVersionRecorded() == "" {
		t.Fatal("catalog version not recorded after materialization")
	}
	// A dependency skill (superpowers) also materialized.
	if _, err := os.Stat(filepath.Join(repo, ".homonto", "catalog", "skills", "brainstorming")); err != nil {
		t.Fatalf("dependency skill brainstorming not materialized: %v", err)
	}
}

func TestApplyRematerializesWhenVersionStale(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(cometTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}
	skillFile := filepath.Join(repo, ".homonto", "catalog", "skills", "comet-open", "SKILL.md")

	// Simulate a partial/stale cache: corrupt content + wipe the recorded version.
	os.WriteFile(skillFile, []byte("STALE"), 0o644)
	e.State.SetCatalogVersion("")
	if err := e.State.Save(e.StateDir); err != nil {
		t.Fatal(err)
	}

	e2 := buildEngine(t, home, repo)
	sets2, _ := e2.Plan()
	if err := e2.Apply(sets2); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if b, _ := os.ReadFile(skillFile); string(b) == "STALE" {
		t.Fatal("stale content not refreshed when recorded version was empty")
	}
}

func TestApplyRematerializesWhenSkillDirMissing(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(cometTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}
	skillDir := filepath.Join(e.CatalogDir(), "comet-open")
	if _, err := os.Stat(skillDir); err != nil {
		t.Fatalf("comet-open not materialized after first apply: %v", err)
	}

	// Delete a materialized skill dir while leaving the recorded catalog
	// version unchanged (still matching the current catalog version).
	if err := os.RemoveAll(skillDir); err != nil {
		t.Fatal(err)
	}

	e2 := buildEngine(t, home, repo)
	sets2, _ := e2.Plan()
	if err := e2.Apply(sets2); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatalf("comet-open not restored after missing dir triggered re-materialization: %v", err)
	}
}

func TestApplySkipsRematerializeWhenVersionMatchesAndDirsIntact(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(cometTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}
	skillDir := filepath.Join(e.CatalogDir(), "comet-open")
	sentinel := filepath.Join(skillDir, "SENTINEL")
	if err := os.WriteFile(sentinel, []byte("keep-me"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Recorded version still matches and all skill dirs are intact, so the
	// re-apply must skip materialization entirely (catalog.Materialize does
	// os.RemoveAll(dstDir) per skill, which would delete the sentinel).
	e2 := buildEngine(t, home, repo)
	sets2, _ := e2.Plan()
	if err := e2.Apply(sets2); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("sentinel removed: re-apply re-materialized despite matching version and intact dirs: %v", err)
	}
}

func TestApplyMaterializesBuiltinCommand(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(commandTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := filepath.Join(repo, ".homonto", "catalog", "commands", "example-command.md")
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("example-command not materialized: %v", err)
	}
	if e.State.CatalogVersionRecorded() == "" {
		t.Fatal("catalog version not recorded after command materialization")
	}
}

func TestApplyRematerializesWhenCommandFileMissing(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(commandTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}
	cmdFile := filepath.Join(e.CommandDir(), "example-command.md")
	if err := os.Remove(cmdFile); err != nil {
		t.Fatal(err)
	}

	e2 := buildEngine(t, home, repo)
	sets2, _ := e2.Plan()
	if err := e2.Apply(sets2); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if _, err := os.Stat(cmdFile); err != nil {
		t.Fatalf("command not restored after missing file triggered re-materialization: %v", err)
	}
}

func TestApplyMaterializesBuiltinSubagent(t *testing.T) {
	e := buildEngineWithSubagent(t)
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p := filepath.Join(e.SubagentDir(), "code-reviewer.md")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("subagent not materialized: %v", err)
	}
}

func TestApplyRematerializesWhenSubagentFileMissing(t *testing.T) {
	e := buildEngineWithSubagent(t)
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	p := filepath.Join(e.SubagentDir(), "code-reviewer.md")
	if err := os.Remove(p); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("subagent not re-materialized when file missing: %v", err)
	}
}
