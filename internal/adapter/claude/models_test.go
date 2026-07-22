package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

// modelBackedCfg is a config whose only model-backed resource is a framework at
// the given scope. homonto no longer derives a default main model from any
// route — an operator who wants a specific main model declares it via
// [settings.claude].model.
func modelBackedCfg(scope string) *config.Config {
	return &config.Config{
		Frameworks: map[string]config.Resource{
			"onto": {Source: "builtin:onto", Scope: scope, Targets: []string{"claude"}},
		},
	}
}

// With no [settings.claude].model and no route-derived default, the adapter
// must NOT synthesize a setting.model key — Claude uses its own default.
func TestDesired_NoMainModelWhenSettingsAbsent(t *testing.T) {
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	c := modelBackedCfg("project")
	got := a.desired(c)
	if v, ok := got["setting.model"]; ok {
		t.Errorf("homongo must not synthesize setting.model when [settings.claude].model is absent, got %q", v)
	}
	gotProj := a.desiredProjectSettings(c)
	if v, ok := gotProj["projsetting.model"]; ok {
		t.Errorf("homonto must not synthesize projsetting.model when [settings.claude].model is absent, got %q", v)
	}
}

// An explicit [settings.claude].model is the only way to project setting.model.
// It lands in the user settings.json; no project-level twin is synthesized.
func TestDesired_ExplicitSettingStillProjects(t *testing.T) {
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	c := modelBackedCfg("project")
	c.Settings = config.Settings{Claude: map[string]any{"model": "sonnet"}}
	got := a.desired(c)
	if got["setting.model"] != `"sonnet"` {
		t.Errorf("explicit [settings.claude].model must project, got %q", got["setting.model"])
	}
	if v, ok := a.desiredProjectSettings(c)["projsetting.model"]; ok {
		t.Errorf("explicit [settings.claude].model must not produce a project-level twin, got %q", v)
	}
}

func TestProjectScopedMCPWritesProjectMCPJSON(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	a := New(home, t.TempDir()).WithProjectRoot(project)
	st, _ := state.Load(t.TempDir())
	c := &config.Config{
		MCPs: map[string]config.MCP{
			"codegraph": {Command: []string{"codegraph", "serve", "--mcp"}, Targets: []string{"claude"}, Scope: "project"},
		},
	}
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(c, cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(project, ".mcp.json"))
	if err != nil {
		t.Fatalf(".mcp.json not written: %v", err)
	}
	doc, _ := jsonutil.Standardize(raw)
	if gjson.GetBytes(doc, "mcpServers.codegraph.command").String() != "codegraph" {
		t.Fatalf("project MCP not projected: %s", doc)
	}
	if gjson.GetBytes(doc, "mcpServers.codegraph.args.0").String() != "serve" {
		t.Fatalf("project MCP args wrong: %s", doc)
	}
	// The user-level .claude.json must not carry it.
	uraw, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if len(uraw) > 0 {
		udoc, _ := jsonutil.Standardize(uraw)
		if gjson.GetBytes(udoc, "mcpServers.codegraph").Exists() {
			t.Fatal("project-scoped MCP leaked into the global .claude.json")
		}
	}
	if _, ok := st.Get("claude", "projmcp.codegraph"); !ok {
		t.Fatal("projmcp.codegraph not recorded")
	}
}
