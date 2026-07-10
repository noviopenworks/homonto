package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
)

// builtinCfg declares one builtin skill directly (explicit [skills] passes
// through ExpandedSkillEntriesForTool unchanged; no framework expansion needed).
func builtinCfg() *config.Config {
	return &config.Config{
		Skills: map[string]config.Resource{
			"brainstorming": {Source: "builtin:brainstorming", Scope: "user", Targets: []string{"claude"}},
		},
	}
}

func TestBuiltinSkillLinksToCatalogRoot(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	// Simulate materialization: the skill dir exists under the catalog root.
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

	dst := filepath.Join(home, ".claude", "skills", "brainstorming")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("skill link missing: %v", err)
	}
	if want := filepath.Join(catalogRoot, "brainstorming"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("claude", "skill.brainstorming"); !ok {
		t.Fatal("skill.brainstorming not recorded in state")
	}

	// Re-plan is a noop for the link.
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
	dst := filepath.Join(home, ".claude", "skills", "brainstorming")

	// Skill removed from config -> delete plan -> link pruned (managed under catalogRoot).
	empty := &config.Config{}
	cs2, err := a.Plan(empty, st)
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

	dst := filepath.Join(home, ".claude", "skills", "brainstorming")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("user file"), 0o644) // a real file, not our link

	_, err := a.Plan(builtinCfg(), st)
	if err == nil {
		t.Fatal("expected conflict for real file at builtin skill link dst")
	}
	if b, _ := os.ReadFile(dst); string(b) != "user file" {
		t.Fatal("conflict clobbered the user file")
	}
}

func builtinCmdCfg() *config.Config {
	return &config.Config{
		Commands: map[string]config.Resource{
			"example-command": {Source: "builtin:example-command", Scope: "user", Targets: []string{"claude"}},
		},
	}
}

func TestBuiltinCommandLinksToCommandCatalogRoot(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	// Simulate materialization: the command file exists under the command root.
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)

	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(builtinCmdCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dst := filepath.Join(home, ".claude", "commands", "example-command.md")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("command link missing: %v", err)
	}
	if want := filepath.Join(cmdRoot, "example-command.md"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("claude", "command.example-command"); !ok {
		t.Fatal("command.example-command not recorded in state")
	}
	// Re-plan is a noop for the link.
	cs2, _ := a.Plan(builtinCmdCfg(), st)
	for _, c := range cs2.Changes {
		if c.Key == "command.example-command" && c.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", c)
		}
	}
}

func TestBuiltinCommandPrunedWhenDeDeclared(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)
	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(builtinCmdCfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".claude", "commands", "example-command.md")

	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatalf("plan empty: %v", err)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared builtin command link not pruned")
	}
}

func TestBuiltinCommandConflictNotClobbered(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)
	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())

	dst := filepath.Join(home, ".claude", "commands", "example-command.md")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("user file"), 0o644)

	if _, err := a.Plan(builtinCmdCfg(), st); err == nil {
		t.Fatal("expected conflict for real file at builtin command link dst")
	}
	if b, _ := os.ReadFile(dst); string(b) != "user file" {
		t.Fatal("conflict clobbered the user file")
	}

	// Apply must fail fast on the same command conflict BEFORE writing any
	// file — not just Plan. a.commands is already populated by the Plan call
	// above (set before its own conflict check), so build a ChangeSet with an
	// unrelated setting write and confirm Apply rejects it without touching
	// settings.json: a partial write (settings.json written, then erroring on
	// the command link) would break the "fail fast before any write" invariant.
	cs := adapter.ChangeSet{Tool: "claude", Changes: []adapter.Change{
		{Action: "create", Key: "setting.foo", New: `"bar"`},
	}}
	if err := a.Apply(cs, resolver(), st); err == nil {
		t.Fatal("expected Apply to fail fast on the command link conflict")
	}
	if _, err := os.Stat(a.settingsJSON()); !os.IsNotExist(err) {
		t.Fatal("Apply wrote settings.json before failing on the command link conflict")
	}
}
