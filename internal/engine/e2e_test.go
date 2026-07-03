package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/tidwall/gjson"
)

const e2eTOML = `
[mcps.codegraph]
command = ["codegraph","serve","--mcp"]

[mcps.brave]
command = ["npx","server-brave"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }
targets = ["claude"]

[skills]
own = ["graphify"]

[settings.claude]
model = "opus"
`

func TestEndToEndApplyIsIdempotent(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(e2eTOML), 0o644)
	os.MkdirAll(filepath.Join(repo, "content", "skills", "graphify"), 0o755)

	build := func() *Engine {
		e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
		if err != nil {
			t.Fatal(err)
		}
		e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "brave-secret", nil }}
		return e
	}

	e := build()
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatal(err)
	}

	// codegraph projected into both tools
	cj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if gjson.GetBytes(cj, "mcpServers.codegraph.command.0").String() != "codegraph" {
		t.Fatal("claude mcp missing")
	}
	oc, _ := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"))
	if gjson.GetBytes(oc, "mcp.codegraph.type").String() != "local" {
		t.Fatal("opencode mcp missing")
	}
	// secret resolved on disk, skill linked
	if gjson.GetBytes(cj, "mcpServers.brave.env.BRAVE_API_KEY").String() != "brave-secret" {
		t.Fatalf("secret not resolved on disk: %s", cj)
	}
	if _, err := os.Lstat(filepath.Join(home, ".claude", "skills", "graphify")); err != nil {
		t.Fatal("claude skill link missing")
	}

	// Second apply: no changes, including the secret-backed MCP.
	e2 := build()
	sets2, _ := e2.Plan()
	if plan.HasChanges(sets2) {
		t.Fatalf("second apply not idempotent: %s", plan.Render(sets2))
	}
}
