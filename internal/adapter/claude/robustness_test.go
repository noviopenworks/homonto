package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

// objKeys returns the immediate member names of the object at root.
func objKeys(t *testing.T, path, root string) map[string]bool {
	t.Helper()
	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := map[string]bool{}
	gjson.GetBytes(doc, root).ForEach(func(k, v gjson.Result) bool {
		out[k.String()] = true
		return true
	})
	return out
}

// TestDottedMCPNameLandsAsLiteralKeyAndConverges reproduces the review's
// path-injection finding: [mcps."corp.internal"] must write the literal
// mcpServers key "corp.internal", not a nested {"corp":{"internal":...}}
// object — and a second plan after apply must be all noop.
func TestDottedMCPNameLandsAsLiteralKeyAndConverges(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{
		MCPs: map[string]config.MCP{
			"corp.internal": {Command: []string{"srv"}, Targets: []string{"claude"}},
		},
		Settings: config.Settings{Claude: map[string]any{"a.b": "v"}},
	}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	mcps := objKeys(t, filepath.Join(home, ".claude.json"), "mcpServers")
	if !mcps["corp.internal"] {
		t.Fatalf("literal key %q missing from mcpServers; got %v", "corp.internal", mcps)
	}
	if mcps["corp"] {
		t.Fatalf("dotted MCP name nested into a %q object: %v", "corp", mcps)
	}
	settings := objKeys(t, filepath.Join(home, ".claude", "settings.json"), "@this")
	if !settings["a.b"] {
		t.Fatalf("literal setting key %q missing; got %v", "a.b", settings)
	}
	if settings["a"] {
		t.Fatalf("dotted setting key nested into an %q object: %v", "a", settings)
	}

	cs2, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("second plan must be all noop (convergence), got %s %s", ch.Action, ch.Key)
		}
	}
}

// TestPluginNameWithSpecialsLandsAsLiteralKey reproduces the review's
// vanishing-write finding: a plugin name like foo@bar.dots must land as a
// literal enabledPlugins key (unescaped, sjson silently drops the write).
func TestPluginNameWithSpecialsLandsAsLiteralKey(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{Plugins: config.Plugins{Claude: []string{"foo@bar.dots"}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	plugins := objKeys(t, filepath.Join(home, ".claude", "settings.json"), "enabledPlugins")
	if !plugins["foo@bar.dots"] {
		t.Fatalf("literal plugin key %q missing from enabledPlugins; got %v", "foo@bar.dots", plugins)
	}

	cs2, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("second plan must be all noop (convergence), got %s %s", ch.Action, ch.Key)
		}
	}
}

// TestForeignSkillSymlinkAborts is the adapter/apply-level guard for the
// release blocker: a symlink in the skills dir pointing OUTSIDE managed content
// (a skill the user linked from their own dotfiles) is user-owned. Plan must
// report a conflict and Apply must abort, leaving the foreign symlink untouched
// — homonto never repoints or removes what it does not own.
func TestForeignSkillSymlinkAborts(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{}`), 0o644)
	content := t.TempDir()
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)

	// The user already linked "onto" to their own directory, outside content.
	foreign := filepath.Join(home, "dotfiles", "onto")
	os.MkdirAll(foreign, 0o755)
	dst := filepath.Join(home, ".claude", "skills", "onto")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(foreign, dst); err != nil {
		t.Fatal(err)
	}

	a := New(home, content)
	st, _ := state.Load(t.TempDir())
	c := &config.Config{Skills: config.Skills{Own: []string{"onto"}}}

	if _, err := a.Plan(c, st); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("plan must conflict on a foreign skill symlink, got %v", err)
	}
	if got, _ := os.Readlink(dst); got != foreign {
		t.Fatalf("plan changed the foreign symlink: now points to %q, want %q", got, foreign)
	}
}

// TestPlanRenderIsDeterministic reproduces the review's map-iteration
// finding: two plans over the same input must render identically.
func TestPlanRenderIsDeterministic(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{
		MCPs: map[string]config.MCP{
			"alpha":   {Command: []string{"a"}, Targets: []string{"claude"}},
			"bravo":   {Command: []string{"b"}, Targets: []string{"claude"}},
			"charlie": {Command: []string{"c"}, Targets: []string{"claude"}},
			"delta":   {Command: []string{"d"}, Targets: []string{"claude"}},
			"echo":    {Command: []string{"e"}, Targets: []string{"claude"}},
			"foxtrot": {Command: []string{"f"}, Targets: []string{"claude"}},
		},
		Settings: config.Settings{Claude: map[string]any{"model": "opus", "theme": "dark"}},
		Skills:   config.Skills{Own: []string{"one", "two", "three"}},
	}
	var first string
	for i := 0; i < 20; i++ {
		cs, err := a.Plan(c, st)
		if err != nil {
			t.Fatalf("plan: %v", err)
		}
		out := plan.Render([]adapter.ChangeSet{cs})
		if i == 0 {
			first = out
			continue
		}
		if out != first {
			t.Fatalf("plan render differs between runs:\n--- run 0 ---\n%s--- run %d ---\n%s", first, i, out)
		}
	}
}

// TestNonObjectRootIsAnError reproduces the review's finding: a managed file
// whose root is valid JSON but not an object (here an array) must be a clear
// error naming the file, not silent downstream corruption.
func TestNonObjectRootIsAnError(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, ".claude.json")
	if err := os.WriteFile(p, []byte(`[]`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	_, err := a.Plan(cfg(), st)
	if err == nil {
		t.Fatal("plan accepted an array-rooted .claude.json; want error")
	}
	if !strings.Contains(err.Error(), p) {
		t.Fatalf("error does not name the file %s: %v", p, err)
	}
	if !strings.Contains(err.Error(), "array") {
		t.Fatalf("error does not name the root kind: %v", err)
	}
}
