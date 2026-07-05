package claude

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
)

// adoptCfg declares a single non-secret MCP so that a pre-existing on-disk
// projection is eligible for adoption (secret keys are never adopted).
func adoptCfg() *config.Config {
	return &config.Config{
		MCPs: map[string]config.MCP{
			"cg": {Command: []string{"codegraph", "serve"}, Env: map[string]string{"MODE": "mcp"}, Targets: []string{"claude"}},
		},
	}
}

// A declared non-secret key already on disk == desired but absent from state
// must be planned as `adopt` (not `noop`); applying it records state and leaves
// the tool file byte-identical (state-only write, no file write).
func TestAdoptRecordsStateWithoutWritingFile(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	c := adoptCfg()

	// Seed the on-disk desired projection via a throwaway apply against a
	// scratch state, so disk exactly equals desired().
	seed, _ := state.Load(t.TempDir())
	cs0, err := a.Plan(c, seed)
	if err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	if err := a.Apply(cs0, resolver(), seed); err != nil {
		t.Fatalf("seed apply: %v", err)
	}

	before, _ := os.ReadFile(filepath.Join(home, ".claude.json"))

	// Plan against an EMPTY state: the key is on disk == desired, not recorded.
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if findChange(cs, "adopt", "mcp.cg") == nil {
		t.Fatalf("expected adopt for mcp.cg, got %+v", cs.Changes)
	}
	if findChange(cs, "noop", "mcp.cg") != nil {
		t.Fatalf("mcp.cg must be adopt, not noop, when absent from state: %+v", cs.Changes)
	}

	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if _, ok := st.Get("claude", "mcp.cg"); !ok {
		t.Fatal("adopt did not record state for mcp.cg")
	}
	after, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if !bytes.Equal(before, after) {
		t.Fatalf("adopt wrote the tool file:\nbefore: %s\nafter:  %s", before, after)
	}
}

// Adoption records the key in state, which makes it visible to pruning: after
// de-declaring the MCP, Plan must yield a delete for the adopted key.
func TestAdoptedKeyIsPruneable(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	c := adoptCfg()

	seed, _ := state.Load(t.TempDir())
	cs0, _ := a.Plan(c, seed)
	if err := a.Apply(cs0, resolver(), seed); err != nil {
		t.Fatalf("seed apply: %v", err)
	}

	// Fresh empty state: adopt the pre-existing key.
	st, _ := state.Load(t.TempDir())
	cs, _ := a.Plan(c, st)
	if findChange(cs, "adopt", "mcp.cg") == nil {
		t.Fatalf("precondition: expected adopt, got %+v", cs.Changes)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// De-declare the MCP → the adopted key must now be pruneable.
	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatalf("re-plan: %v", err)
	}
	if findChange(cs2, "delete", "mcp.cg") == nil {
		t.Fatalf("adopted key not pruneable: %+v", cs2.Changes)
	}
}
