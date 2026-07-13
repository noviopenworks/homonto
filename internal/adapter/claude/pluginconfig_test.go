package claude

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

// pluginConfigCfg declares a claude plugin carrying a config table plus a second
// plugin with no config at all.
func pluginConfigCfg() *config.Config {
	return &config.Config{
		Plugins: config.Plugins{Claude: map[string]config.Plugin{
			"hud":   {Source: "hud@official", Config: map[string]any{"api_endpoint": "https://x", "max_workers": 4}},
			"plain": {Source: "plain@official"},
		}},
	}
}

// TestClaudeProjectsPluginConfig: a plugin's config table is written to
// pluginConfigs[source].options.<k>; a plugin with no config gets no
// pluginConfigs entry.
func TestClaudeProjectsPluginConfig(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{"theme":"dark"}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := pluginConfigCfg()

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	doc, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	if got := gjson.GetBytes(doc, `pluginConfigs.hud@official.options.api_endpoint`); got.String() != "https://x" {
		t.Fatalf("pluginConfigs[hud@official].options.api_endpoint = %q; want https://x", got.String())
	}
	if got := gjson.GetBytes(doc, `pluginConfigs.hud@official.options.max_workers`); got.Int() != 4 {
		t.Fatalf("pluginConfigs[hud@official].options.max_workers = %v; want 4", got.Raw)
	}
	if got := gjson.GetBytes(doc, `pluginConfigs.plain@official`); got.Exists() {
		t.Fatalf("plain@official has no config; want no pluginConfigs entry, got %v", got.Raw)
	}
	if got := gjson.GetBytes(doc, "theme"); got.String() != "dark" {
		t.Fatalf("unrelated key theme = %q; want preserved", got.String())
	}
}

// TestClaudePluginConfigDeDeclared: a pluginConfigs entry recorded in state whose
// config is no longer declared is deleted from settings.json on apply.
func TestClaudePluginConfigDeDeclared(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	// Record the config first.
	c := pluginConfigCfg()
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// De-declare hud's config (keep the plugin itself but drop the config table).
	c2 := &config.Config{Plugins: config.Plugins{Claude: map[string]config.Plugin{
		"hud":   {Source: "hud@official"},
		"plain": {Source: "plain@official"},
	}}}
	cs2, err := a.Plan(c2, st)
	if err != nil {
		t.Fatalf("plan c2: %v", err)
	}
	if findChange(cs2, "delete", "pluginconfig.hud@official") == nil {
		t.Fatalf("expected delete for pluginconfig.hud@official, got %+v", cs2.Changes)
	}
	if err := a.Apply(c2, cs2, resolver(), st); err != nil {
		t.Fatalf("apply c2: %v", err)
	}
	doc, _ := os.ReadFile(settings)
	if got := gjson.GetBytes(doc, `pluginConfigs.hud@official`); got.Exists() {
		t.Fatalf("de-declared config still on disk: %v", got.Raw)
	}
}

// TestClaudeAdoptsPluginConfig: a pre-existing pluginConfigs entry equal to
// desired but absent from state is adopted (recorded into state, file untouched),
// and other unrelated pluginConfigs entries are preserved.
func TestClaudeAdoptsPluginConfig(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	c := &config.Config{Plugins: config.Plugins{Claude: map[string]config.Plugin{
		"hud": {Source: "hud@official", Config: map[string]any{"api_endpoint": "https://x"}},
	}}}

	// Seed disk to exactly equal desired via a throwaway state, plus an
	// unrelated foreign pluginConfigs entry that must survive.
	seed, _ := state.Load(t.TempDir())
	cs0, err := a.Plan(c, seed)
	if err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	if err := a.Apply(c, cs0, resolver(), seed); err != nil {
		t.Fatalf("seed apply: %v", err)
	}
	settings := filepath.Join(home, ".claude", "settings.json")
	raw, _ := os.ReadFile(settings)
	// Inject an unrelated pluginConfigs entry directly.
	withOther, _ := setViaGjson(raw)
	os.WriteFile(settings, withOther, 0o644)
	before, _ := os.ReadFile(settings)

	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if findChange(cs, "adopt", "pluginconfig.hud@official") == nil {
		t.Fatalf("expected adopt for pluginconfig.hud@official, got %+v", cs.Changes)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := st.Get("claude", "pluginconfig.hud@official"); !ok {
		t.Fatal("adopt did not record state for pluginconfig.hud@official")
	}
	after, _ := os.ReadFile(settings)
	if !bytes.Equal(before, after) {
		t.Fatalf("adopt wrote the tool file:\nbefore: %s\nafter:  %s", before, after)
	}
	if got := gjson.GetBytes(after, `pluginConfigs.other@x.options.k`); got.String() != "v" {
		t.Fatalf("unrelated pluginConfigs entry not preserved: %v", got.Raw)
	}
}

// TestClaudePluginConfigPlanDeterministic: after apply, consecutive plans are
// byte-identical (idempotent).
func TestClaudePluginConfigPlanDeterministic(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := pluginConfigCfg()

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
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

// TestClaudePluginConfigWithEnabled: a plugin with BOTH enabled and config
// projects enabledPlugins[source]=true AND pluginConfigs[source].options, and
// after apply the plan is a clean noop with NO setting.enabledPlugins or
// setting.pluginConfigs leaking (the read-back exclusion / idempotency hazard).
func TestClaudePluginConfigWithEnabled(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{Plugins: config.Plugins{Claude: map[string]config.Plugin{
		"hud": {Source: "hud@official", Config: map[string]any{"api_endpoint": "https://x"}},
	}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	doc, _ := os.ReadFile(settings)
	if got := gjson.GetBytes(doc, `enabledPlugins.hud@official`); got.Type != gjson.True {
		t.Fatalf("enabledPlugins[hud@official] = %v; want true", got.Raw)
	}
	if got := gjson.GetBytes(doc, `pluginConfigs.hud@official.options.api_endpoint`); got.String() != "https://x" {
		t.Fatalf("pluginConfigs[hud@official].options.api_endpoint = %q; want https://x", got.String())
	}

	// The next plan must be a clean noop: nothing surfaced as a setting.* key.
	cs2, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan 2: %v", err)
	}
	for _, ch := range cs2.Changes {
		if ch.Action == "noop" {
			continue
		}
		t.Fatalf("second plan is not a clean noop; leaked change %+v (all: %+v)", ch, cs2.Changes)
	}
	if findChange(cs2, "noop", "setting.pluginConfigs") != nil || findChange(cs2, "noop", "setting.enabledPlugins") != nil {
		t.Fatalf("pluginConfigs/enabledPlugins leaked as a setting.* key: %+v", cs2.Changes)
	}
}

// setViaGjson injects an unrelated pluginConfigs.other@x.options.k=v into the
// settings JSON, returning the mutated bytes.
func setViaGjson(doc []byte) ([]byte, error) {
	return jsonutil.SetJSON(doc, "pluginConfigs."+jsonutil.EscapePath("other@x")+".options.k", "v")
}
