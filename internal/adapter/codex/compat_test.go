package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/tomlutil"
)

// seedFixture copies testdata/config.toml into home/.codex/config.toml.
func seedFixture(t *testing.T, home string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// The compatibility-fixture contract: apply projects the declared server,
// preserves unmanaged content, is idempotent, prunes on de-declare while keeping
// unmanaged content, and never clobbers a value homonto does not manage.
func TestCodexCompatibilityFixture(t *testing.T) {
	home := t.TempDir()
	seedFixture(t, home)
	a := New(home)
	st := emptyState(t)
	res := secret.NewResolver()
	cfg := &config.Config{MCPs: map[string]config.MCP{
		"declared": {Command: []string{"declared-server"}, Targets: []string{"codex"}},
	}}
	path := filepath.Join(home, ".codex", "config.toml")

	// 1. apply projects the declared server and preserves unmanaged content.
	cs, err := a.Plan(cfg, st)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(cfg, cs, res, st); err != nil {
		t.Fatal(err)
	}
	doc, _ := os.ReadFile(path)
	if v, ok, _ := tomlutil.Get(doc, "mcp_servers.declared.command"); !ok || v != `"declared-server"` {
		t.Fatalf("declared server not projected: %q ok=%v", v, ok)
	}
	if v, _, _ := tomlutil.Get(doc, "model"); v != `"o3"` {
		t.Fatalf("unmanaged top-level key clobbered: %q", v)
	}
	if v, _, _ := tomlutil.Get(doc, "approval_policy"); v != `"on-request"` {
		t.Fatalf("unmanaged key clobbered: %q", v)
	}
	if v, _, _ := tomlutil.Get(doc, "mcp_servers.user_owned.command"); v != `"user-server"` {
		t.Fatalf("unmanaged server clobbered: %q", v)
	}

	// 2. second apply is idempotent — the file is byte-identical.
	before, _ := os.ReadFile(path)
	cs2, _ := a.Plan(cfg, st)
	for _, c := range cs2.Changes {
		if c.Action != "noop" {
			t.Fatalf("second plan should be all-noop, got %+v", cs2.Changes)
		}
	}
	if err := a.Apply(cfg, cs2, res, st); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("idempotent apply must leave config.toml byte-identical")
	}

	// 3. de-declare prunes the managed server; unmanaged content survives.
	cs3, _ := a.Plan(&config.Config{}, st)
	if err := a.Apply(&config.Config{}, cs3, res, st); err != nil {
		t.Fatal(err)
	}
	doc, _ = os.ReadFile(path)
	if _, ok, _ := tomlutil.Get(doc, "mcp_servers.declared"); ok {
		t.Fatal("de-declared managed server should be pruned")
	}
	if v, _, _ := tomlutil.Get(doc, "mcp_servers.user_owned.command"); v != `"user-server"` {
		t.Fatal("unmanaged server must survive a prune")
	}
	if v, _, _ := tomlutil.Get(doc, "model"); v != `"o3"` {
		t.Fatal("unmanaged top-level key must survive a prune")
	}
}
