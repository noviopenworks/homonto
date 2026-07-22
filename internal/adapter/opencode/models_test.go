package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

// With no [settings.opencode].model and no route-derived default, the adapter
// must NOT synthesize setting.model or setting.small_model keys — OpenCode
// uses its own defaults.
func TestDesiredSettings_NoMainModelWhenSettingsAbsent(t *testing.T) {
	c := &config.Config{
		Frameworks: map[string]config.Resource{
			"onto": {Source: "builtin:onto", Scope: "project", Targets: []string{"opencode"}},
		},
	}
	a := New(t.TempDir(), t.TempDir())
	got := a.desiredSettings(c)
	if v, ok := got["setting.model"]; ok {
		t.Errorf("homonto must not synthesize setting.model when [settings.opencode].model is absent, got %q", v)
	}
	if v, ok := got["setting.small_model"]; ok {
		t.Errorf("homonto must not synthesize setting.small_model when [settings.opencode] is absent, got %q", v)
	}
}

// An explicit [settings.opencode].model is the only way to project
// setting.model. It lands in the global opencode.jsonc.
func TestDesiredSettings_ExplicitSettingStillProjects(t *testing.T) {
	c := &config.Config{
		Frameworks: map[string]config.Resource{
			"onto": {Source: "builtin:onto", Scope: "project", Targets: []string{"opencode"}},
		},
		Settings: config.Settings{OpenCode: map[string]any{"model": "explicit/model"}},
	}
	a := New(t.TempDir(), t.TempDir())
	if got := a.desiredSettings(c); got["setting.model"] != `"explicit/model"` {
		t.Errorf("explicit [settings.opencode].model must project, got %q", got["setting.model"])
	}
}

// projectScopedConfig is a config whose only model-backed resource is a
// project-scoped framework. homonto no longer derives any route-driven keys,
// so desiredProjectSettings is empty regardless of scope.
func projectScopedConfig() *config.Config {
	return &config.Config{
		Frameworks: map[string]config.Resource{
			"onto": {Source: "builtin:onto", Scope: "project", Targets: []string{"opencode"}},
		},
	}
}

func TestDesiredSettings_ProjectScopedNoSynthesizedModelKeys(t *testing.T) {
	c := projectScopedConfig()
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	global := a.desiredSettings(c)
	if _, ok := global["setting.model"]; ok {
		t.Error("project-scoped resources must not synthesize setting.model")
	}
	if _, ok := global["setting.small_model"]; ok {
		t.Error("project-scoped resources must not synthesize setting.small_model")
	}
	proj := a.desiredProjectSettings(c)
	if len(proj) != 0 {
		t.Errorf("no route-derived project settings exist anymore, got %v", proj)
	}
}

func TestDesiredSettings_UserScopedNoSynthesizedModelKeys(t *testing.T) {
	c := projectScopedConfig()
	c.Frameworks["onto"] = config.Resource{Source: "builtin:onto", Scope: "user", Targets: []string{"opencode"}}
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	if got := a.desiredSettings(c); got["setting.model"] != "" || got["setting.small_model"] != "" {
		t.Errorf("a user-scoped framework must not synthesize model keys, got %v", got)
	}
	if proj := a.desiredProjectSettings(c); len(proj) != 0 {
		t.Errorf("no route-derived project settings exist anymore, got %v", proj)
	}
}

// TestProjectScopedMCPAppliesToProjectFileAndMigrates: a scope="project" MCP
// lands in <project>/opencode.jsonc, and one previously applied to the global
// config (recorded as mcp.*) is pruned from it on the next apply.
func TestProjectScopedMCPAppliesToProjectFileAndMigrates(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()

	globalDir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "opencode.jsonc"),
		[]byte(`{"mcp":{"codegraph":{"type":"local","command":["codegraph","serve"],"enabled":true},"keep":{"type":"local","command":["keep"],"enabled":true}}}`), 0o644)
	st, _ := state.Load(t.TempDir())
	st.Set("opencode", "mcp.codegraph", `{"type":"local","command":["codegraph","serve"],"enabled":true}`, "stale")

	a := New(home, t.TempDir()).WithProjectRoot(project)
	c := &config.Config{
		MCPs: map[string]config.MCP{
			"codegraph": {Command: []string{"codegraph", "serve"}, Targets: []string{"opencode"}, Scope: "project"},
		},
	}
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(globalDir, "opencode.jsonc"))
	doc, _ := jsonutil.Standardize(raw)
	if gjson.GetBytes(doc, "mcp.codegraph").Exists() {
		t.Fatalf("project-scoped MCP not pruned from global config: %s", doc)
	}
	if !gjson.GetBytes(doc, "mcp.keep").Exists() {
		t.Fatal("unmanaged global MCP lost")
	}
	praw, err := os.ReadFile(filepath.Join(project, "opencode.jsonc"))
	if err != nil {
		t.Fatalf("project opencode.jsonc not written: %v", err)
	}
	pdoc, _ := jsonutil.Standardize(praw)
	if gjson.GetBytes(pdoc, "mcp.codegraph.type").String() != "local" {
		t.Fatalf("project MCP not projected: %s", pdoc)
	}
	if _, ok := st.Get("opencode", "mcp.codegraph"); ok {
		t.Fatal("stale global mcp.codegraph still recorded")
	}
	if _, ok := st.Get("opencode", "projmcp.codegraph"); !ok {
		t.Fatal("projmcp.codegraph not recorded")
	}

	cs2, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("re-plan: %v", err)
	}
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", ch)
		}
	}
}
