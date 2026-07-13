package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/plan"
	"github.com/noviopenworks/homonto/internal/state"
)

// TestRenderedPlanNeverLeaksSecret exercises the full plan-render path for a
// secret-backed key on both the create and the drift-update branches, asserting
// the rendered text contains neither the resolved value nor the drifted value.
func TestRenderedPlanNeverLeaksSecret(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	// create branch: rendered plan shows the token, not the resolved value
	cs, _ := a.Plan(cfg(), st)
	rendered := plan.Render([]adapter.ChangeSet{cs})
	if strings.Contains(rendered, "SECRET") {
		t.Fatalf("create plan leaked resolved secret:\n%s", rendered)
	}
	if !strings.Contains(rendered, "${pass:ai/brave}") {
		t.Fatalf("create plan should show the unresolved token:\n%s", rendered)
	}

	if err := a.Apply(cfg(), cs, resolver(), st); err != nil {
		t.Fatal(err)
	}

	// drift branch: rendered plan must not contain the drifted on-disk value
	mj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	drift := strings.Replace(string(mj), "SECRET", "DRIFTED-PLAINTEXT", 1)
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(drift), 0o644)

	cs2, _ := a.Plan(cfg(), st)
	rendered2 := plan.Render([]adapter.ChangeSet{cs2})
	if strings.Contains(rendered2, "DRIFTED-PLAINTEXT") {
		t.Fatalf("drift plan leaked the on-disk secret:\n%s", rendered2)
	}
	if !strings.Contains(rendered2, adapter.SecretRedaction) {
		t.Fatalf("drift plan should redact the old value:\n%s", rendered2)
	}
}

// TestStateAbsentDiskValueIsRedacted covers unknown provenance: the key is not
// in state (fresh/lost state.json) but the on-disk value may be a previously
// applied resolved secret — plan must never print it as Old.
func TestStateAbsentDiskValueIsRedacted(t *testing.T) {
	home := t.TempDir()
	disk := `{"mcpServers":{"brave":{"type":"stdio","command":"npx","args":["server-brave"],"env":{"K":"sk-previously-applied-secret"}}}}`
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(disk), 0o600)
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir()) // empty state: provenance unknown

	literalCfg := &config.Config{
		MCPs: map[string]config.MCP{
			"brave": {Command: []string{"npx", "server-brave"}, Env: map[string]string{"K": "literal"}, Targets: []string{"claude"}},
		},
	}
	cs, err := a.Plan(literalCfg, st)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range cs.Changes {
		if c.Key != "mcp.brave" {
			continue
		}
		found = true
		if c.Action != "update" {
			t.Fatalf("expected update, got %s", c.Action)
		}
		if strings.Contains(c.Old, "sk-previously-applied-secret") {
			t.Fatalf("state-absent plan leaked the on-disk value in Old: %q", c.Old)
		}
		if c.Old != adapter.SecretRedaction {
			t.Fatalf("unknown-provenance Old should be redacted, got %q", c.Old)
		}
	}
	if !found {
		t.Fatal("expected an mcp.brave change")
	}
}

// TestSecretToLiteralTransitionRedacts covers the case where a key that was a
// secret reference is edited to a literal: the on-disk value is still a resolved
// secret, so plan output must not print it.
func TestSecretToLiteralTransitionRedacts(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	// apply the secret-backed MCP (disk now holds resolved "SECRET")
	cs, _ := a.Plan(cfg(), st)
	if err := a.Apply(cfg(), cs, resolver(), st); err != nil {
		t.Fatal(err)
	}

	// user edits the env value to a literal (no ${...})
	literalCfg := &config.Config{
		MCPs: map[string]config.MCP{
			"brave": {Command: []string{"npx", "server-brave"}, Env: map[string]string{"K": "now-literal"}, Targets: []string{"claude"}},
		},
	}
	cs2, _ := a.Plan(literalCfg, st)
	rendered := plan.Render([]adapter.ChangeSet{cs2})
	if strings.Contains(rendered, "SECRET") {
		t.Fatalf("secret->literal transition leaked the resolved secret:\n%s", rendered)
	}
}
