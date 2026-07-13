package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

func findChange(cs adapter.ChangeSet, action adapter.Action, key string) *adapter.Change {
	for i, c := range cs.Changes {
		if c.Action == action && c.Key == key {
			return &cs.Changes[i]
		}
	}
	return nil
}

// Deep review CRITICAL: a de-declared MCP stayed in opencode.jsonc forever —
// no delete action existed. It must be planned as a delete, pruned from disk,
// and garbage-collected from state.
func TestOpenCodeRemovedMCPIsPruned(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{
		MCPs: map[string]config.MCP{"codegraph": {Command: []string{"codegraph", "serve"}, Targets: []string{"opencode"}}},
	}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatal(err)
	}

	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatal(err)
	}
	del := findChange(cs2, "delete", "mcp.codegraph")
	if del == nil {
		t.Fatalf("plan lacks a delete for the de-declared MCP: %+v", cs2.Changes)
	}
	if del.Old != adapter.SecretRedaction {
		t.Fatalf("delete Old must always be redacted (stale provenance), got %q", del.Old)
	}
	if err := a.Apply(cs2, noSecret(), st); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"))
	doc, _ := jsonutil.Standardize(raw)
	if gjson.GetBytes(doc, "mcp.codegraph").Exists() {
		t.Fatalf("de-declared MCP still on disk: %s", doc)
	}
	if _, ok := st.Get("opencode", "mcp.codegraph"); ok {
		t.Fatal("state still records the removed MCP")
	}
}

// A de-declared plugin must be removed from the plugin array while unmanaged
// elements survive.
func TestOpenCodeRemovedPluginIsRemovedFromArray(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(`{"plugin":["existing"]}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{Plugins: config.Plugins{OpenCode: map[string]config.Plugin{"quota": {Source: "@x/quota"}}}}

	cs, _ := a.Plan(c, st)
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatal(err)
	}

	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatal(err)
	}
	if findChange(cs2, "delete", "plugin.@x/quota") == nil {
		t.Fatalf("plan lacks a delete for the de-declared plugin: %+v", cs2.Changes)
	}
	if err := a.Apply(cs2, noSecret(), st); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "opencode.jsonc"))
	doc, _ := jsonutil.Standardize(raw)
	arr := gjson.GetBytes(doc, "plugin").Array()
	if len(arr) != 1 || arr[0].String() != "existing" {
		t.Fatalf("plugin array after prune = %s", gjson.GetBytes(doc, "plugin").Raw)
	}
	if _, ok := st.Get("opencode", "plugin.@x/quota"); ok {
		t.Fatal("state still records the removed plugin")
	}
}

// Removing a skill from skills.own must prune its symlink here too.
func TestOpenCodeRemovedSkillLinkIsPruned(t *testing.T) {
	home := t.TempDir()
	content := t.TempDir()
	os.MkdirAll(filepath.Join(content, "skills", "foo"), 0o755)
	a := New(home, content)
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(cfgWithSkills("user", "foo"), st)
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(home, ".config", "opencode", "skills", "foo")
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
	if err := a.Apply(cs2, noSecret(), st); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("dangling skill symlink not removed")
	}
	if _, ok := st.Get("opencode", "skill.foo"); ok {
		t.Fatal("state still records the removed skill")
	}
}
