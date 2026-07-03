package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
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

	if err := a.Apply(cs, resolver(), st); err != nil {
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
