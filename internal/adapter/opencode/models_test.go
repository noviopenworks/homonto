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

func TestDesiredSettings_ModelRoutesProjectDefaults(t *testing.T) {
	c := &config.Config{
		Models: config.ModelConfig{
			OpenCode: map[string]config.ModelRoute{
				"architectural": {Model: "openai/gpt-5.6-terra"},
				"coding":        {Model: "openai/gpt-5.6-mini"},
				"trivial":       {Model: "openai/gpt-5.6-nano"},
			},
		},
	}
	a := New(t.TempDir(), t.TempDir())
	got := a.desiredSettings(c)
	if got["setting.model"] != `"openai/gpt-5.6-terra"` {
		t.Errorf("architectural route must project setting.model, got %q", got["setting.model"])
	}
	if got["setting.small_model"] != `"openai/gpt-5.6-nano"` {
		t.Errorf("trivial route must project setting.small_model, got %q", got["setting.small_model"])
	}
}

func TestDesiredSettings_ExplicitSettingWinsOverRoute(t *testing.T) {
	c := &config.Config{
		Settings: config.Settings{OpenCode: map[string]any{"model": "explicit/model"}},
		Models: config.ModelConfig{
			OpenCode: map[string]config.ModelRoute{
				"architectural": {Model: "route/model"},
			},
		},
	}
	a := New(t.TempDir(), t.TempDir())
	if got := a.desiredSettings(c); got["setting.model"] != `"explicit/model"` {
		t.Errorf("explicit [settings.opencode].model must win, got %q", got["setting.model"])
	}
}

func TestDesiredSettings_NoRoutesNoModelKeys(t *testing.T) {
	a := New(t.TempDir(), t.TempDir())
	got := a.desiredSettings(&config.Config{})
	if _, ok := got["setting.model"]; ok {
		t.Error("no model routes must not synthesize setting.model")
	}
	if _, ok := got["setting.small_model"]; ok {
		t.Error("no model routes must not synthesize setting.small_model")
	}
}

// projectScopedConfig is a config whose only model-backed resource is a
// project-scoped framework — the case whose routes must land in the
// project-level opencode.jsonc, not the global one.
func projectScopedConfig() *config.Config {
	return &config.Config{
		Frameworks: map[string]config.Resource{
			"onto": {Source: "builtin:onto", Scope: "project", Targets: []string{"opencode"}},
		},
		Models: config.ModelConfig{
			OpenCode: map[string]config.ModelRoute{
				"architectural": {Model: "zai/glm-big"},
				"coding":        {Model: "zai/glm-big"},
				"trivial":       {Model: "zai/glm-small"},
			},
		},
	}
}

func TestDesiredSettings_ProjectScopedRoutesLeaveGlobal(t *testing.T) {
	c := projectScopedConfig()
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	global := a.desiredSettings(c)
	if _, ok := global["setting.model"]; ok {
		t.Error("project-scoped routes must not project the global setting.model")
	}
	if _, ok := global["setting.small_model"]; ok {
		t.Error("project-scoped routes must not project the global setting.small_model")
	}
	proj := a.desiredProjectSettings(c)
	if proj["projsetting.model"] != `"zai/glm-big"` {
		t.Errorf("architectural route must project projsetting.model, got %q", proj["projsetting.model"])
	}
	if proj["projsetting.small_model"] != `"zai/glm-small"` {
		t.Errorf("trivial route must project projsetting.small_model, got %q", proj["projsetting.small_model"])
	}
}

func TestDesiredSettings_UserScopedResourceKeepsRoutesGlobal(t *testing.T) {
	c := projectScopedConfig()
	c.Frameworks["onto"] = config.Resource{Source: "builtin:onto", Scope: "user", Targets: []string{"opencode"}}
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	if got := a.desiredSettings(c); got["setting.model"] != `"zai/glm-big"` {
		t.Errorf("a user-scoped framework must keep setting.model global, got %q", got["setting.model"])
	}
	if proj := a.desiredProjectSettings(c); len(proj) != 0 {
		t.Errorf("a user-scoped framework must project no project-level settings, got %v", proj)
	}
}

func TestDesiredSettings_NoProjectRootKeepsRoutesGlobal(t *testing.T) {
	c := projectScopedConfig()
	a := New(t.TempDir(), t.TempDir()) // no project root
	if got := a.desiredSettings(c); got["setting.model"] != `"zai/glm-big"` {
		t.Errorf("without a project root the routes must stay global, got %q", got["setting.model"])
	}
	if proj := a.desiredProjectSettings(c); len(proj) != 0 {
		t.Errorf("without a project root there must be no project-level settings, got %v", proj)
	}
}

// projectCmdCfg is a config whose only model-backed resource is a
// project-scoped builtin command, with all three routes declared.
func projectCmdCfg() *config.Config {
	return &config.Config{
		Commands: map[string]config.Resource{
			"example-command": {Source: "builtin:example-command", Scope: "project", Targets: []string{"opencode"}},
		},
		Models: config.ModelConfig{
			OpenCode: map[string]config.ModelRoute{
				"architectural": {Model: "zai/glm-big"},
				"coding":        {Model: "zai/glm-big"},
				"trivial":       {Model: "zai/glm-small"},
			},
		},
	}
}

// TestProjectScopedRoutesApplyToProjectFileAndMigrate covers the leak fix end
// to end: a fully project-scoped config lands its default-model keys in
// <project>/opencode.jsonc, and keys previously applied to the global config
// (recorded as setting.*) are pruned from it on the next apply.
func TestProjectScopedRoutesApplyToProjectFileAndMigrate(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)

	// Seed the pre-fix world: the global config carries the leaked keys and
	// state records them under setting.*.
	globalDir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "opencode.jsonc"),
		[]byte(`{"model":"zai/glm-big","small_model":"zai/glm-small","theme":"keep"}`), 0o644)
	st, _ := state.Load(t.TempDir())
	st.Set("opencode", "setting.model", `"zai/glm-big"`, "stale")
	st.Set("opencode", "setting.small_model", `"zai/glm-small"`, "stale")

	a := New(home, t.TempDir()).WithProjectRoot(project).WithCommandCatalogRoot(cmdRoot)
	c := projectCmdCfg()
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(globalDir, "opencode.jsonc"))
	doc, _ := jsonutil.Standardize(raw)
	if gjson.GetBytes(doc, "model").Exists() || gjson.GetBytes(doc, "small_model").Exists() {
		t.Fatalf("leaked global model keys not pruned: %s", doc)
	}
	if gjson.GetBytes(doc, "theme").String() != "keep" {
		t.Fatal("unmanaged global key lost")
	}
	praw, err := os.ReadFile(filepath.Join(project, "opencode.jsonc"))
	if err != nil {
		t.Fatalf("project opencode.jsonc not written: %v", err)
	}
	pdoc, _ := jsonutil.Standardize(praw)
	if gjson.GetBytes(pdoc, "model").String() != "zai/glm-big" {
		t.Fatalf("project model = %s", pdoc)
	}
	if gjson.GetBytes(pdoc, "small_model").String() != "zai/glm-small" {
		t.Fatalf("project small_model = %s", pdoc)
	}
	if _, ok := st.Get("opencode", "setting.model"); ok {
		t.Fatal("stale global setting.model still recorded")
	}
	if _, ok := st.Get("opencode", "projsetting.model"); !ok {
		t.Fatal("projsetting.model not recorded")
	}

	// Steady state: the second plan must be all-noop.
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

func TestDesiredProjectSettings_ExplicitSettingSuppressesProjectTwin(t *testing.T) {
	c := projectScopedConfig()
	c.Settings = config.Settings{OpenCode: map[string]any{"model": "explicit/model"}}
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	proj := a.desiredProjectSettings(c)
	// A project-level model would override the explicit global one in
	// OpenCode's merge order, silently inverting "explicit wins".
	if _, ok := proj["projsetting.model"]; ok {
		t.Error("explicit [settings.opencode].model must suppress the project-level route key")
	}
	if proj["projsetting.small_model"] != `"zai/glm-small"` {
		t.Errorf("small_model has no explicit override and must still project, got %q", proj["projsetting.small_model"])
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
