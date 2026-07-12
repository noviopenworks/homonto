package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/noviopenworks/homonto/internal/tomlutil"
)

func emptyState(t *testing.T) *state.State {
	t.Helper()
	st, err := state.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func cfgWithCodexMCP() *config.Config {
	return &config.Config{
		MCPs: map[string]config.MCP{
			"demo": {Command: []string{"npx", "-y", "demo-server"}, Env: map[string]string{"K": "v"}, Targets: []string{"codex"}},
			// a claude-only MCP must NOT project to codex
			"other": {Command: []string{"other"}, Targets: []string{"claude"}},
		},
	}
}

func configTOML(t *testing.T, home string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	return b
}

func TestCodexProjectsMCP(t *testing.T) {
	home := t.TempDir()
	a := New(home)
	st := emptyState(t)
	res := secret.NewResolver()

	cs, err := a.Plan(cfgWithCodexMCP(), st)
	if err != nil {
		t.Fatal(err)
	}
	if cs.Tool != "codex" {
		t.Fatalf("tool = %q", cs.Tool)
	}
	// exactly one change (the codex-targeted MCP), a create
	if len(cs.Changes) != 1 || cs.Changes[0].Action != "create" || cs.Changes[0].Key != "mcp.demo" {
		t.Fatalf("unexpected changes: %+v", cs.Changes)
	}
	if err := a.Apply(cs, res, st); err != nil {
		t.Fatal(err)
	}
	doc := configTOML(t, home)
	if v, ok := tomlutil.Get(doc, "mcp_servers.demo.command"); !ok || v != `"npx"` {
		t.Fatalf("command not projected: %q ok=%v", v, ok)
	}
	if v, _ := tomlutil.Get(doc, "mcp_servers.demo.args"); v != `["-y","demo-server"]` {
		t.Fatalf("args wrong: %q", v)
	}
	if v, _ := tomlutil.Get(doc, "mcp_servers.demo.env"); v != `{"K":"v"}` {
		t.Fatalf("env wrong: %q", v)
	}
	// the claude-only MCP must not be present
	if _, ok := tomlutil.Get(doc, "mcp_servers.other"); ok {
		t.Fatal("a non-codex MCP must not project to codex")
	}

	// second plan is a noop (idempotent)
	cs2, _ := a.Plan(cfgWithCodexMCP(), st)
	if len(cs2.Changes) != 1 || cs2.Changes[0].Action != "noop" {
		t.Fatalf("want noop, got %+v", cs2.Changes)
	}
}

func TestCodexPrunesDeDeclared(t *testing.T) {
	home := t.TempDir()
	a := New(home)
	st := emptyState(t)
	res := secret.NewResolver()
	cs, _ := a.Plan(cfgWithCodexMCP(), st)
	if err := a.Apply(cs, res, st); err != nil {
		t.Fatal(err)
	}
	// de-declare all MCPs
	empty := &config.Config{}
	cs2, _ := a.Plan(empty, st)
	if len(cs2.Changes) != 1 || cs2.Changes[0].Action != "delete" {
		t.Fatalf("want delete, got %+v", cs2.Changes)
	}
	if err := a.Apply(cs2, res, st); err != nil {
		t.Fatal(err)
	}
	if _, ok := tomlutil.Get(configTOML(t, home), "mcp_servers.demo"); ok {
		t.Fatal("de-declared server should be pruned")
	}
}

func TestCodexPlanDoesNotRevealSecret(t *testing.T) {
	home := t.TempDir()
	a := New(home)
	st := emptyState(t)
	c := &config.Config{MCPs: map[string]config.MCP{
		"demo": {Command: []string{"srv"}, Env: map[string]string{"TOKEN": "${pass:demo/token}"}, Targets: []string{"codex"}},
	}}
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Changes) != 1 {
		t.Fatalf("want 1 change, got %+v", cs.Changes)
	}
	ch := cs.Changes[0]
	// New keeps the unresolved token (never a resolved value); Old is empty for a
	// create. Neither field may contain a resolved secret.
	if ch.New == "" || !containsToken(ch.New) {
		t.Fatalf("New should carry the unresolved token, got %q", ch.New)
	}
}

func containsToken(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '$' && s[i+1] == '{' {
			return true
		}
	}
	return false
}
