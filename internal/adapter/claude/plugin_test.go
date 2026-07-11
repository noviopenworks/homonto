package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

// pluginCfg declares one enabled and one disabled claude plugin, source-keyed.
func pluginCfg() *config.Config {
	dis := false
	return &config.Config{
		Plugins: config.Plugins{Claude: map[string]config.Plugin{
			"hud": {Source: "hud@official"},
			"off": {Source: "off@official", Enabled: &dis},
		}},
	}
}

// TestClaudeProjectsPluginEnableDisable: an enabled plugin writes
// enabledPlugins[source]=true, a disabled one writes enabledPlugins[source]=false
// (a managed value, not absence), and unrelated settings are preserved.
func TestClaudeProjectsPluginEnableDisable(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{"theme":"dark"}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := pluginCfg()

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	doc, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	if got := gjson.GetBytes(doc, `enabledPlugins.hud@official`); !got.Exists() || got.Type != gjson.True {
		t.Fatalf("enabledPlugins[hud@official] = %v; want true", got.Raw)
	}
	if got := gjson.GetBytes(doc, `enabledPlugins.off@official`); !got.Exists() || got.Type != gjson.False {
		t.Fatalf("enabledPlugins[off@official] = %v; want false (disabled emits a managed false)", got.Raw)
	}
	if got := gjson.GetBytes(doc, "theme"); got.String() != "dark" {
		t.Fatalf("unrelated key theme = %q; want preserved", got.String())
	}
}

// TestClaudePluginPlanIsDeterministic: after apply, two consecutive plan
// renders are byte-identical (idempotent, stable order).
func TestClaudePluginPlanIsDeterministic(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := pluginCfg()

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var first string
	for i := 0; i < 5; i++ {
		cs, err := a.Plan(c, st)
		if err != nil {
			t.Fatalf("plan %d: %v", i, err)
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
