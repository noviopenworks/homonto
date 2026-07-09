package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

func findChange(cs adapter.ChangeSet, action, key string) *adapter.Change {
	for i, c := range cs.Changes {
		if c.Action == action && c.Key == key {
			return &cs.Changes[i]
		}
	}
	return nil
}

// Deep review CRITICAL: removing [mcps.brave] from homonto.toml left
// mcpServers.brave (with resolved secrets) in ~/.claude.json forever — there
// was no delete action anywhere. De-declared keys must be planned as deletes,
// pruned from disk, and garbage-collected from state.
func TestClaudeRemovedMCPIsPruned(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"keep":true}`), 0o644)
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	cs, err := a.Plan(cfg(), st)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}

	// De-declare everything: brave (and setting.model) leave the config.
	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatal(err)
	}
	del := findChange(cs2, "delete", "mcp.brave")
	if del == nil {
		t.Fatalf("plan lacks a delete for the de-declared MCP: %+v", cs2.Changes)
	}
	if del.Old != adapter.SecretRedaction {
		t.Fatalf("delete Old must always be redacted (stale provenance), got %q", del.Old)
	}
	if findChange(cs2, "delete", "setting.model") == nil {
		t.Fatalf("plan lacks a delete for the de-declared setting: %+v", cs2.Changes)
	}

	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatal(err)
	}
	mj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if gjson.GetBytes(mj, "mcpServers.brave").Exists() {
		t.Fatalf("de-declared MCP still on disk: %s", mj)
	}
	if !gjson.GetBytes(mj, "keep").Bool() {
		t.Fatal("unmanaged key lost during prune")
	}
	sj, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if gjson.GetBytes(sj, "model").Exists() {
		t.Fatalf("de-declared setting still on disk: %s", sj)
	}
	if _, ok := st.Get("claude", "mcp.brave"); ok {
		t.Fatal("state still records the removed MCP")
	}
	if _, ok := st.Get("claude", "setting.model"); ok {
		t.Fatal("state still records the removed setting")
	}
}

// A key still declared must never be planned as a delete merely because it is
// missing from disk — that is drift (a create), not de-declaration.
func TestClaudeDriftIsNotMistakenForOrphan(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(cfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}
	// Wipe the tool file out-of-band; config still declares everything.
	os.Remove(filepath.Join(home, ".claude.json"))

	cs2, err := a.Plan(cfg(), st)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs2.Changes {
		if c.Action == "delete" {
			t.Fatalf("declared-but-missing key planned as delete: %+v", c)
		}
	}
}

// Removing a skill from skills.own must prune its symlink; before the fix the
// link dangled forever because skill links were never recorded in state.
func TestClaudeRemovedSkillLinkIsPruned(t *testing.T) {
	home := t.TempDir()
	content := t.TempDir()
	os.MkdirAll(filepath.Join(content, "skills", "foo"), 0o755)
	a := New(home, content)
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(cfgWithSkills("user", "foo"), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(home, ".claude", "skills", "foo")
	if _, err := os.Lstat(dst); err != nil {
		t.Fatalf("precondition: skill link missing: %v", err)
	}

	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatal(err)
	}
	if findChange(cs2, "delete", "skill.foo") == nil {
		t.Fatalf("plan lacks a delete for the removed skill: %+v", cs2.Changes)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("dangling skill symlink not removed")
	}
	if _, ok := st.Get("claude", "skill.foo"); ok {
		t.Fatal("state still records the removed skill")
	}
}

// Pruning must never delete something homonto does not own: a non-symlink at
// the skill path is a conflict error, and the file survives untouched.
func TestClaudePruneNeverRemovesNonHomontoFile(t *testing.T) {
	home := t.TempDir()
	content := t.TempDir()
	os.MkdirAll(filepath.Join(content, "skills", "foo"), 0o755)
	a := New(home, content)
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(cfgWithSkills("user", "foo"), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}
	// The user replaces the link with their own file, then de-declares foo.
	dst := filepath.Join(home, ".claude", "skills", "foo")
	os.Remove(dst)
	os.WriteFile(dst, []byte("user data"), 0o644)

	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatal(err)
	}
	if findChange(cs2, "delete", "skill.foo") == nil {
		t.Fatalf("plan lacks a delete for the removed skill: %+v", cs2.Changes)
	}
	if err := a.Apply(cs2, resolver(), st); err == nil {
		t.Fatal("expected a conflict error, not silent removal of a user file")
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "user data" {
		t.Fatalf("user file destroyed by prune: %q %v", got, err)
	}
}
