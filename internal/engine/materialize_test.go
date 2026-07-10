package engine

import (
	"os"
	"path/filepath"
	"testing"

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

func buildEngine(t *testing.T, home, repo string) *Engine {
	t.Helper()
	e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
	return e
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
