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

func boolPtr(b bool) *bool { return &b }

// marketplaceCfg declares a github marketplace plus a directory marketplace.
func marketplaceCfg() *config.Config {
	return &config.Config{
		Marketplaces: config.Marketplaces{Claude: map[string]config.Marketplace{
			"official": {Source: "github", Repo: "anthropics/claude-plugins"},
			"local":    {Source: "directory", Path: "/opt/mkt"},
		}},
	}
}

// TestClaudeProjectsMarketplace: a github marketplace projects
// extraKnownMarketplaces[name].source={source,repo}; a directory marketplace
// projects {source,path}; autoUpdate is absent when auto_update is unset.
func TestClaudeProjectsMarketplace(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{"theme":"dark"}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := marketplaceCfg()

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
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.official.source.source`); got.String() != "github" {
		t.Fatalf("official.source.source = %q; want github", got.String())
	}
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.official.source.repo`); got.String() != "anthropics/claude-plugins" {
		t.Fatalf("official.source.repo = %q; want anthropics/claude-plugins", got.String())
	}
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.official.autoUpdate`); got.Exists() {
		t.Fatalf("official.autoUpdate present without auto_update: %v", got.Raw)
	}
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.local.source.source`); got.String() != "directory" {
		t.Fatalf("local.source.source = %q; want directory", got.String())
	}
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.local.source.path`); got.String() != "/opt/mkt" {
		t.Fatalf("local.source.path = %q; want /opt/mkt", got.String())
	}
	if got := gjson.GetBytes(doc, "theme"); got.String() != "dark" {
		t.Fatalf("unrelated key theme = %q; want preserved", got.String())
	}
}

// TestClaudeMarketplaceAutoUpdate: auto_update=true is projected as autoUpdate:true.
func TestClaudeMarketplaceAutoUpdate(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{Marketplaces: config.Marketplaces{Claude: map[string]config.Marketplace{
		"official": {Source: "github", Repo: "anthropics/claude-plugins", AutoUpdate: boolPtr(true)},
	}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	doc, _ := os.ReadFile(settings)
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.official.autoUpdate`); got.Type != gjson.True {
		t.Fatalf("official.autoUpdate = %v; want true", got.Raw)
	}
}

// TestClaudeMarketplaceDeDeclared: a marketplace recorded in state and no longer
// declared is deleted from settings.json on apply.
func TestClaudeMarketplaceDeDeclared(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	c := marketplaceCfg()
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Drop the "local" marketplace.
	c2 := &config.Config{Marketplaces: config.Marketplaces{Claude: map[string]config.Marketplace{
		"official": {Source: "github", Repo: "anthropics/claude-plugins"},
	}}}
	cs2, err := a.Plan(c2, st)
	if err != nil {
		t.Fatalf("plan c2: %v", err)
	}
	if findChange(cs2, "delete", "marketplace.local") == nil {
		t.Fatalf("expected delete for marketplace.local, got %+v", cs2.Changes)
	}
	if err := a.Apply(c2, cs2, resolver(), st); err != nil {
		t.Fatalf("apply c2: %v", err)
	}
	doc, _ := os.ReadFile(settings)
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.local`); got.Exists() {
		t.Fatalf("de-declared marketplace still on disk: %v", got.Raw)
	}
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.official`); !got.Exists() {
		t.Fatalf("still-declared marketplace was removed: %v", got.Raw)
	}
}

// TestClaudeAdoptsMarketplace: a pre-existing extraKnownMarketplaces entry equal
// to desired but absent from state is adopted (recorded, file untouched), and an
// unrelated foreign marketplace entry survives.
func TestClaudeAdoptsMarketplace(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	c := &config.Config{Marketplaces: config.Marketplaces{Claude: map[string]config.Marketplace{
		"official": {Source: "github", Repo: "anthropics/claude-plugins"},
	}}}

	// Seed disk to exactly equal desired via a throwaway state.
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
	// Inject an unrelated foreign marketplace entry that must survive.
	withOther, _ := jsonutil.SetJSON(raw, "extraKnownMarketplaces."+jsonutil.EscapePath("other")+".source.source", "directory")
	os.WriteFile(settings, withOther, 0o644)
	before, _ := os.ReadFile(settings)

	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if findChange(cs, "adopt", "marketplace.official") == nil {
		t.Fatalf("expected adopt for marketplace.official, got %+v", cs.Changes)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := st.Get("claude", "marketplace.official"); !ok {
		t.Fatal("adopt did not record state for marketplace.official")
	}
	after, _ := os.ReadFile(settings)
	if !bytes.Equal(before, after) {
		t.Fatalf("adopt wrote the tool file:\nbefore: %s\nafter:  %s", before, after)
	}
	if got := gjson.GetBytes(after, `extraKnownMarketplaces.other.source.source`); got.String() != "directory" {
		t.Fatalf("unrelated marketplace entry not preserved: %v", got.Raw)
	}
}

// TestClaudeMarketplacePlanDeterministic: after apply, consecutive plans are
// byte-identical (idempotent).
func TestClaudeMarketplacePlanDeterministic(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := marketplaceCfg()

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

// TestClaudeFourNamespaceIdempotency: a settings.json carrying a plain setting,
// a plugin (enabledPlugins), a pluginConfig, AND a marketplace re-plans to a
// clean noop with NO namespace leaking as a setting.* key (the four-namespace
// read-back exclusion hazard).
func TestClaudeFourNamespaceIdempotency(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	settings := filepath.Join(home, ".claude", "settings.json")
	os.WriteFile(settings, []byte(`{}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{
		Settings: config.Settings{Claude: map[string]any{"theme": "dark"}},
		Plugins: config.Plugins{Claude: map[string]config.Plugin{
			"hud": {Source: "hud@official", Config: map[string]any{"api_endpoint": "https://x"}},
		}},
		Marketplaces: config.Marketplaces{Claude: map[string]config.Marketplace{
			"official": {Source: "github", Repo: "anthropics/claude-plugins"},
		}},
	}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	doc, _ := os.ReadFile(settings)
	if got := gjson.GetBytes(doc, `extraKnownMarketplaces.official.source.repo`); got.String() != "anthropics/claude-plugins" {
		t.Fatalf("marketplace not written: %v", got.Raw)
	}

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
	if findChange(cs2, "noop", "setting.extraKnownMarketplaces") != nil ||
		findChange(cs2, "noop", "setting.pluginConfigs") != nil ||
		findChange(cs2, "noop", "setting.enabledPlugins") != nil {
		t.Fatalf("a managed namespace leaked as a setting.* key: %+v", cs2.Changes)
	}
}
