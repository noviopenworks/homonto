package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/tidwall/gjson"
)

// A malformed opencode.jsonc must not block the Claude adapter (tool-adapters
// spec: "that adapter aborts and reports ... while other tools still proceed").
func TestUnparseableToolFileDoesNotBlockOtherTool(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(`
[mcps.codegraph]
command = ["codegraph","serve"]
targets = ["claude"]
`), 0o644)
	// broken JSONC that Standardize cannot parse
	ocDir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(ocDir, 0o755)
	os.WriteFile(filepath.Join(ocDir, "opencode.jsonc"), []byte(`{ "plugin": [ this is not json `), 0o644)

	e, err := Build(context.Background(), filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Resolver = &secret.Resolver{Getenv: func(string) string { return "" }, Pass: func(string) (string, error) { return "", nil }}

	sets, err := e.Plan()
	if err != nil {
		t.Fatalf("plan should not hard-fail on one bad tool: %v", err)
	}
	if len(e.Warnings) == 0 {
		t.Fatal("expected a warning about the unparseable opencode file")
	}
	if err := e.Apply(context.Background(), sets); err != nil {
		t.Fatalf("apply should proceed for the healthy tool: %v", err)
	}
	cj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if gjson.GetBytes(cj, "mcpServers.codegraph.command").String() != "codegraph" {
		t.Fatalf("claude was blocked by the broken opencode file: %s", cj)
	}
}
