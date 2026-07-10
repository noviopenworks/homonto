package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

func builtinCfg() *config.Config {
	return &config.Config{
		Skills: map[string]config.Resource{
			"brainstorming": {Source: "builtin:brainstorming", Scope: "user", Targets: []string{"opencode"}},
		},
	}
}

func resolver() *secret.Resolver {
	return &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
}

func TestBuiltinSkillLinksToCatalogRoot(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())

	cs, err := a.Plan(builtinCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".config", "opencode", "skills", "brainstorming")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("skill link missing: %v", err)
	}
	if want := filepath.Join(catalogRoot, "brainstorming"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("opencode", "skill.brainstorming"); !ok {
		t.Fatal("skill.brainstorming not recorded")
	}
	cs2, _ := a.Plan(builtinCfg(), st)
	for _, c := range cs2.Changes {
		if c.Key == "skill.brainstorming" && c.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", c)
		}
	}
}

func TestBuiltinSkillPrunedWhenDeDeclared(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())
	cs, _ := a.Plan(builtinCfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".config", "opencode", "skills", "brainstorming")
	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatalf("plan empty: %v", err)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared builtin skill link not pruned")
	}
}

func TestBuiltinSkillConflictNotClobbered(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())
	dst := filepath.Join(home, ".config", "opencode", "skills", "brainstorming")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("user file"), 0o644)
	if _, err := a.Plan(builtinCfg(), st); err == nil {
		t.Fatal("expected conflict for real file at builtin skill link dst")
	}
	if b, _ := os.ReadFile(dst); string(b) != "user file" {
		t.Fatal("conflict clobbered the user file")
	}
}
