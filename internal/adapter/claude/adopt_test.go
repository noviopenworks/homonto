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

// Phantom-drift case: a recorded non-secret key whose on-disk value was changed
// out of band to a NEW value, and whose desired config was rebuilt to that same
// new value, leaves state's Applied stale (still the old value's hash). Disk ==
// desired, so the old code emitted a bare `noop` that never refreshes Applied —
// so ObserveHashes(disk) != Applied reports the key as drifted forever. The fix:
// a true noop requires Applied == hash(disk); otherwise adopt, refreshing the
// hash and clearing the phantom drift.
func TestStaleAppliedRefreshedViaAdopt(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())

	// 1. Record setting.model=opus in the real state (Applied = hash of "opus").
	st, _ := state.Load(t.TempDir())
	c1 := &config.Config{Settings: config.Settings{Claude: map[string]any{"model": "opus"}}}
	cs1, err := a.Plan(c1, st)
	if err != nil {
		t.Fatalf("plan c1: %v", err)
	}
	if err := a.Apply(cs1, resolver(), st); err != nil {
		t.Fatalf("apply c1: %v", err)
	}

	// 2. Out-of-band: change the ON-DISK value to "sonnet" WITHOUT touching the
	//    real state — apply the new desired against a throwaway scratch state that
	//    shares the same home. Real st keeps Applied = hash("opus").
	scratch, _ := state.Load(t.TempDir())
	c2 := &config.Config{Settings: config.Settings{Claude: map[string]any{"model": "sonnet"}}}
	csScratch, err := a.Plan(c2, scratch)
	if err != nil {
		t.Fatalf("plan scratch: %v", err)
	}
	if err := a.Apply(csScratch, resolver(), scratch); err != nil {
		t.Fatalf("apply scratch: %v", err)
	}

	// Precondition: state's Applied is now stale vs disk (phantom drift).
	obs, err := a.ObserveHashes(st)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	e, _ := st.Get("claude", "setting.model")
	if obs["setting.model"] == e.Applied {
		t.Fatal("precondition: Applied should be stale vs disk before the fix")
	}

	// 3. Plan the (now-matching) desired c2 against the real state: disk == desired
	//    but Applied is stale, so this must be `adopt`, not `noop`.
	cs, err := a.Plan(c2, st)
	if err != nil {
		t.Fatalf("plan c2: %v", err)
	}
	if findChange(cs, "adopt", "setting.model") == nil {
		t.Fatalf("expected adopt for stale-Applied setting.model, got %+v", cs.Changes)
	}
	if findChange(cs, "noop", "setting.model") != nil {
		t.Fatalf("setting.model must be adopt, not noop, when Applied is stale: %+v", cs.Changes)
	}

	before, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply c2: %v", err)
	}

	// Applied is refreshed to the on-disk hash; the tool file is byte-unchanged.
	obs2, _ := a.ObserveHashes(st)
	e2, _ := st.Get("claude", "setting.model")
	if e2.Applied != obs2["setting.model"] {
		t.Fatalf("adopt did not refresh Applied: %q != %q", e2.Applied, obs2["setting.model"])
	}
	after, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if !bytes.Equal(before, after) {
		t.Fatalf("adopt wrote the tool file:\nbefore: %s\nafter:  %s", before, after)
	}

	// Drift is cleared: a second Plan now yields noop.
	cs3, err := a.Plan(c2, st)
	if err != nil {
		t.Fatalf("re-plan: %v", err)
	}
	if findChange(cs3, "noop", "setting.model") == nil {
		t.Fatalf("expected noop after refresh, got %+v", cs3.Changes)
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
