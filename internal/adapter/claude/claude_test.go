package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

func cfg() *config.Config {
	return &config.Config{
		MCPs: map[string]config.MCP{
			"brave": {Command: []string{"npx", "server-brave"}, Env: map[string]string{"K": "${pass:ai/brave}"}, Targets: []string{"claude"}},
		},
		Settings: config.Settings{Claude: map[string]any{"model": "opus"}},
	}
}

func resolver() *secret.Resolver {
	return &secret.Resolver{
		Getenv: os.Getenv,
		Pass:   func(string) (string, error) { return "SECRET", nil },
	}
}

func TestPlanThenApplyIsSurgicalAndIdempotent(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"keep":true,"mcpServers":{}}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"theme":"dark"}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	cs, err := a.Plan(cfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(cs.Changes) == 0 {
		t.Fatal("expected changes on first plan")
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	mj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if !gjson.GetBytes(mj, "keep").Bool() {
		t.Fatal("unmanaged .claude.json key lost")
	}
	if gjson.GetBytes(mj, "mcpServers.brave.env.K").String() != "SECRET" {
		t.Fatalf("secret not resolved on apply: %s", mj)
	}
	sj, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if gjson.GetBytes(sj, "theme").String() != "dark" {
		t.Fatal("unmanaged settings key lost")
	}
	if gjson.GetBytes(sj, "model").String() != "opus" {
		t.Fatal("managed setting not written")
	}

	// second plan = no changes (idempotent), including the secret-backed MCP.
	cs2, _ := a.Plan(cfg(), st)
	for _, c := range cs2.Changes {
		if c.Action != "noop" {
			t.Fatalf("expected idempotent noop, got %+v", c)
		}
	}
}

func TestSecretWithSpecialCharsDoesNotCorruptFile(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	res := &secret.Resolver{
		Getenv: os.Getenv,
		Pass:   func(string) (string, error) { return `x","injected":"y`, nil },
	}
	cs, _ := a.Plan(cfg(), st)
	if err := a.Apply(cs, res, st); err != nil {
		t.Fatalf("apply with quote-bearing secret: %v", err)
	}
	mj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	if !gjson.ValidBytes(mj) {
		t.Fatalf("apply produced invalid JSON: %s", mj)
	}
	if gjson.GetBytes(mj, "mcpServers.brave.env.K").String() != `x","injected":"y` {
		t.Fatalf("secret not stored verbatim as a string: %s", mj)
	}
	if gjson.GetBytes(mj, "mcpServers.brave.injected").Exists() || gjson.GetBytes(mj, "injected").Exists() {
		t.Fatal("secret value injected a sibling key")
	}
	// still idempotent
	cs2, _ := a.Plan(cfg(), st)
	for _, c := range cs2.Changes {
		if c.Action != "noop" {
			t.Fatalf("not idempotent with special-char secret: %+v", c)
		}
	}
}

func TestStateHasNoPlaintextSecret(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	dir := t.TempDir()
	st, _ := state.Load(dir)

	cs, _ := a.Plan(cfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if strings.Contains(string(raw), "SECRET") {
		t.Fatalf("state.json leaked resolved secret: %s", raw)
	}
	if !strings.Contains(string(raw), "${pass:ai/brave}") {
		t.Fatalf("state.json should keep the unresolved token: %s", raw)
	}
}

func TestSecretDriftPlanIsRedacted(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	// first apply records hashed state
	cs, _ := a.Plan(cfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatal(err)
	}

	// simulate out-of-band drift of the secret value on disk
	mj, _ := os.ReadFile(filepath.Join(home, ".claude.json"))
	mj2 := strings.Replace(string(mj), "SECRET", "LEAKED-DRIFT-VALUE", 1)
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(mj2), 0o644)

	cs2, _ := a.Plan(cfg(), st)
	var found bool
	for _, c := range cs2.Changes {
		if c.Key == "mcp.brave" {
			found = true
			if c.Action != "update" {
				t.Fatalf("expected drift update, got %s", c.Action)
			}
			if strings.Contains(c.Old, "LEAKED-DRIFT-VALUE") {
				t.Fatalf("drift plan leaked the on-disk secret in Old: %q", c.Old)
			}
			if c.Old != "«secret»" {
				t.Fatalf("secret Old should be redacted, got %q", c.Old)
			}
		}
	}
	if !found {
		t.Fatal("expected an mcp.brave change after drift")
	}
}

func TestSkillsOnlyConfigPlansAndAppliesLinks(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{}`), 0o644)
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{}`), 0o644)

	content := t.TempDir()
	os.MkdirAll(filepath.Join(content, "skills", "onto"), 0o755)

	a := New(home, content)
	st, _ := state.Load(t.TempDir())
	c := &config.Config{Skills: config.Skills{Own: []string{"onto"}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	nonNoop := 0
	for _, ch := range cs.Changes {
		if ch.Action != "noop" {
			nonNoop++
		}
	}
	if nonNoop == 0 {
		t.Fatal("skills-only config: plan must contain a non-noop change for the missing link")
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".claude", "skills", "onto")
	if got, err := os.Readlink(dst); err != nil || got != filepath.Join(content, "skills", "onto") {
		t.Fatalf("link not created: %v %s", err, got)
	}

	cs2, _ := a.Plan(c, st)
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("second plan must be all noop, got %s %s", ch.Action, ch.Key)
		}
	}
}
