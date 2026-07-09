package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
)

// observeCfg exercises every recorded prefix: mcp.*, setting.*, plugin.*, skill.*.
func observeCfg() *config.Config {
	c := cfgWithSkills("user", "onto")
	c.MCPs = map[string]config.MCP{"codegraph": {Command: []string{"codegraph", "serve"}, Targets: []string{"claude"}}}
	c.Settings = config.Settings{Claude: map[string]any{"model": "opus"}}
	c.Plugins = config.Plugins{Claude: []string{"repo-plugin"}}
	return c
}

func applyObserveCfg(t *testing.T, home, content string) (*Adapter, *state.State) {
	t.Helper()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)

	a := New(home, content)
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(observeCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	return a, st
}

// TestClaudeObserveHashesUnchangedEqualsApplied is the load-bearing correctness
// test: for every recorded key still on disk, ObserveHashes must reproduce the
// exact hash apply stored in Entry.Applied.
func TestClaudeObserveHashesUnchangedEqualsApplied(t *testing.T) {
	home, content := t.TempDir(), t.TempDir()
	a, st := applyObserveCfg(t, home, content)

	obs, err := a.ObserveHashes(st)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	keys := st.Keys("claude")
	sawMCP, sawSetting, sawPlugin, sawSkill := false, false, false, false
	for _, key := range keys {
		e, _ := st.Get("claude", key)
		h, ok := obs[key]
		if !ok {
			t.Fatalf("recorded key %q missing from ObserveHashes", key)
		}
		if h != e.Applied {
			t.Fatalf("key %q: observed %q != Applied %q", key, h, e.Applied)
		}
		switch {
		case key == "mcp.codegraph":
			sawMCP = true
		case key == "setting.model":
			sawSetting = true
		case key == "plugin.repo-plugin":
			sawPlugin = true
		case key == "skill.onto":
			sawSkill = true
		}
	}
	if !(sawMCP && sawSetting && sawPlugin && sawSkill) {
		t.Fatalf("expected mcp/setting/plugin/skill keys exercised, got %v", keys)
	}
}

func TestClaudeObserveHashesEditedDiffers(t *testing.T) {
	home, content := t.TempDir(), t.TempDir()
	a, st := applyObserveCfg(t, home, content)
	before, _ := a.ObserveHashes(st)

	// Out-of-band edit of a managed setting value.
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"),
		[]byte(`{"model":"sonnet","enabledPlugins":{"repo-plugin":true}}`), 0o644)

	after, err := a.ObserveHashes(st)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if after["setting.model"] == before["setting.model"] {
		t.Fatalf("edited setting.model hash unchanged: %q", after["setting.model"])
	}
	e, _ := st.Get("claude", "setting.model")
	if after["setting.model"] == e.Applied {
		t.Fatalf("edited setting.model still equals Applied")
	}
}

func TestClaudeObserveHashesAbsentOmitted(t *testing.T) {
	home, content := t.TempDir(), t.TempDir()
	a, st := applyObserveCfg(t, home, content)

	// Remove a managed MCP from disk entirely.
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)

	obs, err := a.ObserveHashes(st)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if _, ok := obs["mcp.codegraph"]; ok {
		t.Fatalf("deleted mcp.codegraph must be omitted, got %q", obs["mcp.codegraph"])
	}
	// Remove the skill symlink; it too must drop out.
	os.Remove(filepath.Join(home, ".claude", "skills", "onto"))
	obs2, _ := a.ObserveHashes(st)
	if _, ok := obs2["skill.onto"]; ok {
		t.Fatal("removed skill.onto link must be omitted")
	}
}
