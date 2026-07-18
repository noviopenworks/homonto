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

// routedCfg is a config whose only model-backed resource is a framework at the
// given scope, with the architectural route declared for claude.
func routedCfg(scope string) *config.Config {
	return &config.Config{
		Frameworks: map[string]config.Resource{
			"onto": {Source: "builtin:onto", Scope: scope, Targets: []string{"claude"}},
		},
		Models: config.ModelConfig{
			Claude: map[string]config.ModelRoute{
				"architectural": {Model: "opus"},
				"coding":        {Model: "sonnet"},
				"trivial":       {Model: "haiku"},
			},
		},
	}
}

func TestDesired_ProjectScopedRouteLeavesUserSettings(t *testing.T) {
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	c := routedCfg("project")
	if got := a.desired(c); got["setting.model"] != "" {
		t.Errorf("project-scoped routes must not project the user setting.model, got %q", got["setting.model"])
	}
	if got := a.desiredProjectSettings(c); got["projsetting.model"] != `"opus"` {
		t.Errorf("architectural route must project projsetting.model, got %q", got["projsetting.model"])
	}
}

func TestDesired_UserScopedRouteStaysGlobal(t *testing.T) {
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	c := routedCfg("user")
	if got := a.desired(c); got["setting.model"] != `"opus"` {
		t.Errorf("a user-scoped framework must keep setting.model global, got %q", got["setting.model"])
	}
	if got := a.desiredProjectSettings(c); len(got) != 0 {
		t.Errorf("a user-scoped framework must project no project-level settings, got %v", got)
	}
}

func TestDesired_ExplicitSettingSuppressesProjectTwin(t *testing.T) {
	a := New(t.TempDir(), t.TempDir()).WithProjectRoot(t.TempDir())
	c := routedCfg("project")
	c.Settings = config.Settings{Claude: map[string]any{"model": "sonnet"}}
	if got := a.desired(c); got["setting.model"] != `"sonnet"` {
		t.Errorf("explicit [settings.claude].model must win, got %q", got["setting.model"])
	}
	// A project-level model would override the explicit user one in Claude's
	// settings merge order, silently inverting "explicit wins".
	if got := a.desiredProjectSettings(c); len(got) != 0 {
		t.Errorf("explicit [settings.claude].model must suppress the project-level twin, got %v", got)
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
