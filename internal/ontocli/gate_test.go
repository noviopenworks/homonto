package ontocli

import (
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

func gateIDs(gs []pendingGate) []string {
	var ids []string
	for _, g := range gs {
		ids = append(ids, g.ID)
	}
	return ids
}

func TestPendingGates_ByPhaseAndState(t *testing.T) {
	// design with no isolation → the isolation gate.
	if ids := gateIDs(pendingGates("c", ontostate.State{Phase: "design"})); len(ids) != 1 || ids[0] != "isolation" {
		t.Errorf("design/no-isolation gates = %v, want [isolation]", ids)
	}
	// design with isolation set → no gate.
	if ids := gateIDs(pendingGates("c", ontostate.State{Phase: "design", Isolation: "branch"})); len(ids) != 0 {
		t.Errorf("design/isolation-set gates = %v, want none", ids)
	}
	// build with nothing recorded → build-mode + tdd-mode.
	if ids := gateIDs(pendingGates("c", ontostate.State{Phase: "build"})); strings.Join(ids, ",") != "build-mode,tdd-mode" {
		t.Errorf("build gates = %v, want [build-mode tdd-mode]", ids)
	}
	// full close missing everything → merged, guides, integration.
	full := ontostate.State{Phase: "close", Workflow: "full"}
	if ids := gateIDs(pendingGates("c", full)); strings.Join(ids, ",") != "close-merged,guides,integration" {
		t.Errorf("full close gates = %v, want [close-merged guides integration]", ids)
	}
	// a tweak close does not gate guides.
	tweak := ontostate.State{Phase: "close", Workflow: "tweak", Close: ontostate.Close{Merged: true}, Integration: "merge"}
	if ids := gateIDs(pendingGates("c", tweak)); len(ids) != 0 {
		t.Errorf("resolved tweak close gates = %v, want none", ids)
	}
}

func TestGateCommand_JSONAndHuman(t *testing.T) {
	dir := prepWorkspace(t)
	seedCloseState(t, dir, ontostate.State{
		Change: "demo", Workflow: "full", Phase: "close", Created: "2026-07-10",
		Verify: ontostate.Verify{Result: "pass"}, Close: ontostate.Close{Merged: true}, Guides: "updated",
		Integration: "", // the only pending gate
	})

	// JSON carries the structured schema a dialog renders.
	jout, err := runOnto(t, "gate", "demo", "--dir", dir, "--json")
	if err != nil {
		t.Fatalf("gate --json: %v", err)
	}
	for _, want := range []string{`"id": "integration"`, `"set_command"`, `"merge"`, `"pr"`} {
		if !strings.Contains(jout, want) {
			t.Errorf("gate --json missing %q:\n%s", want, jout)
		}
	}
	// Human form names the set command.
	hout, err := runOnto(t, "gate", "demo", "--dir", dir)
	if err != nil {
		t.Fatalf("gate: %v", err)
	}
	if !strings.Contains(hout, "onto set integration demo") {
		t.Errorf("gate human output missing set command:\n%s", hout)
	}
}
