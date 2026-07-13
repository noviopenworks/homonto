package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

// Real Claude Code mcpServers entries look like:
//
//	{"type": "stdio", "command": "npx", "args": ["-y", "x"], "env": {...}}
//
// command is a STRING and args a separate array — never a command array.

func TestDesiredEmitsRealClaudeMCPSchema(t *testing.T) {
	a := New(t.TempDir(), t.TempDir())
	want := a.desired(cfg())["mcp.brave"]

	v := gjson.Parse(want)
	if got := v.Get("type").String(); got != "stdio" {
		t.Fatalf("type must be %q, got %q in %s", "stdio", got, want)
	}
	cmd := v.Get("command")
	if cmd.Type != gjson.String {
		t.Fatalf("command must be a JSON string, got %s in %s", cmd.Type, want)
	}
	if cmd.String() != "npx" {
		t.Fatalf("command must be the executable, got %q", cmd.String())
	}
	args := v.Get("args")
	if !args.IsArray() {
		t.Fatalf("args must be an array, got %s in %s", args.Type, want)
	}
	if got := args.Array(); len(got) != 1 || got[0].String() != "server-brave" {
		t.Fatalf("args must carry the command tail, got %s", args.Raw)
	}
	if v.Get("env.K").String() != "${pass:ai/brave}" {
		t.Fatalf("env must be preserved unresolved, got %s", want)
	}
}

func TestDesiredOmitsEmptyArgsAndEnv(t *testing.T) {
	a := New(t.TempDir(), t.TempDir())
	c := &config.Config{MCPs: map[string]config.MCP{
		"cg": {Command: []string{"codegraph"}, Targets: []string{"claude"}},
	}}
	want := a.desired(c)["mcp.cg"]
	v := gjson.Parse(want)
	if v.Get("args").Exists() {
		t.Fatalf("args key must be omitted when empty: %s", want)
	}
	if v.Get("env").Exists() {
		t.Fatalf("env key must be omitted when empty: %s", want)
	}
	if v.Get("type").String() != "stdio" || v.Get("command").String() != "codegraph" {
		t.Fatalf("minimal server must still be type=stdio + string command: %s", want)
	}
}

func TestApplyOntoRealClaudeJSONPreservesSchema(t *testing.T) {
	home := t.TempDir()
	fixture, err := os.ReadFile(filepath.Join("testdata", "real-claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), fixture, 0o644); err != nil {
		t.Fatal(err)
	}

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(cfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cfg(), cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	mj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	// The managed server must land in the real schema.
	brave := gjson.GetBytes(mj, "mcpServers.brave")
	if brave.Get("type").String() != "stdio" {
		t.Fatalf("managed server missing type=stdio: %s", brave.Raw)
	}
	if brave.Get("command").Type != gjson.String {
		t.Fatalf("managed server command must be a string: %s", brave.Raw)
	}
	if !brave.Get("args").IsArray() {
		t.Fatalf("managed server args must be an array: %s", brave.Raw)
	}
	// The pre-existing real-schema server must be untouched.
	cg := gjson.GetBytes(mj, "mcpServers.codegraph")
	if cg.Get("command").String() != "codegraph" || cg.Get("command").Type != gjson.String {
		t.Fatalf("unmanaged server command mangled: %s", cg.Raw)
	}
	if got := cg.Get("args").Array(); len(got) != 2 || got[0].String() != "serve" || got[1].String() != "--mcp" {
		t.Fatalf("unmanaged server args mangled: %s", cg.Raw)
	}
	// Unmanaged top-level keys survive.
	if gjson.GetBytes(mj, "numStartups").Int() != 42 || !gjson.GetBytes(mj, "hasCompletedOnboarding").Bool() {
		t.Fatalf("unmanaged top-level keys lost: %s", mj)
	}
}
