package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/tidwall/gjson"
)

// tuiPath is the second managed file: ~/.config/opencode/tui.json under home.
func tuiPath(home string) string {
	return filepath.Join(home, ".config", "opencode", "tui.json")
}

// TestOpenCodeTUICreatesFileWithTheme: a config with only [tui.opencode]
// theme="gruvbox" and NO tui.json on disk projects the theme into a freshly
// created ~/.config/opencode/tui.json.
func TestOpenCodeTUICreatesFileWithTheme(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{TUI: config.TUI{OpenCode: map[string]any{"theme": "gruvbox"}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	raw, err := os.ReadFile(tuiPath(home))
	if err != nil {
		t.Fatalf("first apply did not create tui.json: %v", err)
	}
	if got := gjson.GetBytes(raw, "theme").String(); got != "gruvbox" {
		t.Fatalf("tui.json theme = %q; want \"gruvbox\" (%s)", got, raw)
	}
}

// TestOpenCodeTUIPreservesUnrelatedKey: an unmanaged key already in tui.json
// survives the projection of a declared key.
func TestOpenCodeTUIPreservesUnrelatedKey(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(tuiPath(home), []byte(`{"font":"mono"}`), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{TUI: config.TUI{OpenCode: map[string]any{"theme": "gruvbox"}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	raw, _ := os.ReadFile(tuiPath(home))
	if got := gjson.GetBytes(raw, "font").String(); got != "mono" {
		t.Fatalf("unmanaged tui.json key lost: font = %q (%s)", got, raw)
	}
	if got := gjson.GetBytes(raw, "theme").String(); got != "gruvbox" {
		t.Fatalf("declared tui key not projected: theme = %q (%s)", got, raw)
	}
}

// TestOpenCodeTUIPrunesDeDeclaredKey: a tui key that was applied and recorded in
// state, then removed from config, is de-declared — the next apply deletes it
// from tui.json.
func TestOpenCodeTUIPrunesDeDeclaredKey(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())

	// First apply records tui.theme in state and writes tui.json.
	c1 := &config.Config{TUI: config.TUI{OpenCode: map[string]any{"theme": "gruvbox"}}}
	cs1, err := a.Plan(c1, st)
	if err != nil {
		t.Fatalf("plan1: %v", err)
	}
	if err := a.Apply(cs1, noSecret(), st); err != nil {
		t.Fatalf("apply1: %v", err)
	}

	// Now the config declares no tui keys → tui.theme is de-declared.
	c2 := &config.Config{}
	cs2, err := a.Plan(c2, st)
	if err != nil {
		t.Fatalf("plan2: %v", err)
	}
	sawDelete := false
	for _, ch := range cs2.Changes {
		if ch.Key == "tui.theme" && ch.Action == "delete" {
			sawDelete = true
		}
	}
	if !sawDelete {
		t.Fatalf("expected a delete for de-declared tui.theme, got %+v", cs2.Changes)
	}
	if err := a.Apply(cs2, noSecret(), st); err != nil {
		t.Fatalf("apply2: %v", err)
	}
	raw, _ := os.ReadFile(tuiPath(home))
	if gjson.GetBytes(raw, "theme").Exists() {
		t.Fatalf("de-declared tui.theme not pruned from tui.json: %s", raw)
	}
}

// TestOpenCodeTUIAdoptLeavesFileByteIdentical: a pre-existing tui.json key that
// already matches the declared value, with empty state, is adopted — recorded
// into state without rewriting tui.json (byte-for-byte identical).
func TestOpenCodeTUIAdoptLeavesFileByteIdentical(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	original := "{\n  // hand-written tui config\n  \"theme\": \"gruvbox\"\n}\n"
	os.WriteFile(tuiPath(home), []byte(original), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir()) // empty state → matching key yields adopt
	c := &config.Config{TUI: config.TUI{OpenCode: map[string]any{"theme": "gruvbox"}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	sawAdopt := false
	for _, ch := range cs.Changes {
		if ch.Key == "tui.theme" {
			if ch.Action != "adopt" {
				t.Fatalf("expected adopt for pre-existing matching tui.theme, got %s", ch.Action)
			}
			sawAdopt = true
		}
	}
	if !sawAdopt {
		t.Fatal("expected an adopt for tui.theme")
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, _ := os.ReadFile(tuiPath(home))
	if string(got) != original {
		t.Fatalf("adopt rewrote tui.json.\nwant: %q\ngot:  %q", original, string(got))
	}
	// The adopt must have recorded state so a second plan is a steady-state noop.
	cs2, _ := a.Plan(c, st)
	for _, ch := range cs2.Changes {
		if ch.Key == "tui.theme" && ch.Action != "noop" {
			t.Fatalf("second plan after adopt should be noop, got %s", ch.Action)
		}
	}
}

// TestOpenCodeTUIOnlyLeavesOpencodeJsoncByteIdentical proves two-file
// independence: a config with ONLY [tui.opencode] keys must not rewrite
// opencode.jsonc — its hand-written JSONC comments survive untouched.
func TestOpenCodeTUIOnlyLeavesOpencodeJsoncByteIdentical(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	original := "{\n  // keep me: a real JSONC comment\n  \"theme\": \"x\"\n}\n"
	cfgPath := filepath.Join(dir, "opencode.jsonc")
	os.WriteFile(cfgPath, []byte(original), 0o644)

	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{TUI: config.TUI{OpenCode: map[string]any{"theme": "gruvbox"}}}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// tui.json got the theme...
	tui, err := os.ReadFile(tuiPath(home))
	if err != nil {
		t.Fatalf("tui.json not created: %v", err)
	}
	if gjson.GetBytes(tui, "theme").String() != "gruvbox" {
		t.Fatalf("tui.json theme not projected: %s", tui)
	}
	// ...while opencode.jsonc is byte-for-byte unchanged.
	got, _ := os.ReadFile(cfgPath)
	if string(got) != original {
		t.Fatalf("tui-only apply rewrote opencode.jsonc.\nwant: %q\ngot:  %q", original, string(got))
	}
}

// TestOpenCodeTUIAndSettingsIdempotent: a config with both [settings.opencode]
// and [tui.opencode] applies once, then a second plan reports no changes.
func TestOpenCodeTUIAndSettingsIdempotent(t *testing.T) {
	home := t.TempDir()
	a := New(home, t.TempDir())
	st, _ := state.Load(t.TempDir())
	c := &config.Config{
		Settings: config.Settings{OpenCode: map[string]any{"x": "y"}},
		TUI:      config.TUI{OpenCode: map[string]any{"theme": "z"}},
	}

	cs, err := a.Plan(c, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	cs2, _ := a.Plan(c, st)
	for _, ch := range cs2.Changes {
		if ch.Action != "noop" {
			t.Fatalf("settings+tui config not idempotent: %+v", ch)
		}
	}
}
