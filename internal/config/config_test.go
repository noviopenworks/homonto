package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const sample = `
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]

[mcps.brave]
command = ["npx", "-y", "server-brave"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }
targets = ["claude"]

[skills]
own = ["graphify", "comet"]

[plugins]
claude = ["claude-hud@official"]
opencode = ["@slkiser/opencode-quota"]

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"
`

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "homonto.toml")
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := c.MCPs["codegraph"].Command; len(got) != 3 || got[0] != "codegraph" {
		t.Fatalf("codegraph command = %v", got)
	}
	if got := c.MCPs["brave"].Env["BRAVE_API_KEY"]; got != "${pass:ai/brave}" {
		t.Fatalf("brave env = %q", got)
	}
	if got := c.MCPs["codegraph"].TargetsOrAll(); len(got) != 2 {
		t.Fatalf("default targets = %v", got)
	}
	if got := c.MCPs["brave"].TargetsOrAll(); len(got) != 1 || got[0] != "claude" {
		t.Fatalf("brave targets = %v", got)
	}
	if c.Settings.Claude["model"] != "opus" {
		t.Fatalf("claude model = %v", c.Settings.Claude["model"])
	}
	if len(c.Skills.Own) != 2 {
		t.Fatalf("skills = %v", c.Skills.Own)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestLoadRejectsIndexLikeNames reproduces the verify round's corruption
// finding: sjson treats all-digit keys ("0") and "-" + digits ("-1") as array
// indices, so [mcps."0"] silently turns mcpServers into a JSON ARRAY. Empty
// names address nothing. All such names must be a clear load-time error for
// every key homonto writes into a tool file.
func TestLoadRejectsIndexLikeNames(t *testing.T) {
	bad := []struct{ label, doc, name string }{
		{"mcp empty", "[mcps.\"\"]\ncommand = [\"x\"]\n", ""},
		{"mcp zero", "[mcps.\"0\"]\ncommand = [\"x\"]\n", "0"},
		{"mcp minus-one", "[mcps.\"-1\"]\ncommand = [\"x\"]\n", "-1"},
		{"claude setting", "[settings.claude]\n\"0\" = \"x\"\n", "0"},
		{"opencode setting", "[settings.opencode]\n\"-1\" = \"x\"\n", "-1"},
		{"claude plugin", "[plugins]\nclaude = [\"7\"]\n", "7"},
		{"opencode plugin", "[plugins]\nopencode = [\"\"]\n", ""},
	}
	for _, tc := range bad {
		p := filepath.Join(t.TempDir(), "homonto.toml")
		if err := os.WriteFile(p, []byte(tc.doc), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(p)
		if err == nil {
			t.Fatalf("%s: name %q accepted; want load error", tc.label, tc.name)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.name)) {
			t.Fatalf("%s: error does not name the entry %q: %v", tc.label, tc.name, err)
		}
	}
	good := []string{"corp.internal", "a0", "0a", "v2", "-x1"}
	for _, name := range good {
		p := filepath.Join(t.TempDir(), "homonto.toml")
		doc := "[mcps." + strconv.Quote(name) + "]\ncommand = [\"x\"]\n"
		if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(p); err != nil {
			t.Fatalf("valid name %q rejected: %v", name, err)
		}
	}
}

// TestLoadRejectsBadSkillNames reproduces the review's traversal finding:
// own = ["../../../escaped"] must be a load-time error, not a symlink
// planted outside $HOME. Every non-bare-directory-name entry is rejected.
func TestLoadRejectsBadSkillNames(t *testing.T) {
	for _, bad := range []string{"../evil", "..", ".", "", "a/b", `a\b`, "/abs"} {
		dir := t.TempDir()
		p := filepath.Join(dir, "homonto.toml")
		doc := "[skills]\nown = [" + strconv.Quote(bad) + "]\n"
		if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(p)
		if err == nil {
			t.Fatalf("skill name %q accepted; want load error", bad)
		}
		if !strings.Contains(err.Error(), strconv.Quote(bad)) {
			t.Fatalf("error for %q does not name the entry: %v", bad, err)
		}
	}
}

// TestLoadSkillScope covers the per-project-skills capability: [skills] scope
// selects where owned skills install. Absent/empty normalizes to "user"
// (back-compat); "project" is honored; any other value is a named load error.
func TestLoadSkillScope(t *testing.T) {
	loadScope := func(doc string) (string, error) {
		p := filepath.Join(t.TempDir(), "homonto.toml")
		if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		c, err := Load(p)
		if err != nil {
			return "", err
		}
		return c.Skills.Scope, nil
	}

	// Absent scope defaults to "user".
	if got, err := loadScope("[skills]\nown = [\"a\"]\n"); err != nil || got != "user" {
		t.Fatalf("absent scope = %q, %v; want \"user\", nil", got, err)
	}
	// Explicit empty also normalizes to "user".
	if got, err := loadScope("[skills]\nscope = \"\"\nown = [\"a\"]\n"); err != nil || got != "user" {
		t.Fatalf("empty scope = %q, %v; want \"user\", nil", got, err)
	}
	// Explicit user and project honored.
	if got, err := loadScope("[skills]\nscope = \"user\"\n"); err != nil || got != "user" {
		t.Fatalf("user scope = %q, %v", got, err)
	}
	if got, err := loadScope("[skills]\nscope = \"project\"\n"); err != nil || got != "project" {
		t.Fatalf("project scope = %q, %v", got, err)
	}
	// Invalid value is rejected, naming the offending value and the valid set.
	err := loadDoc(t, "[skills]\nscope = \"global\"\n")
	if err == nil {
		t.Fatal("scope \"global\" accepted; want load error")
	}
	for _, want := range []string{strconv.Quote("global"), "user", "project"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %v does not mention %q", err, want)
		}
	}
}

func loadDoc(t *testing.T, doc string) error {
	t.Helper()
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(p)
	return err
}

// TestLoadRejectsUnknownTargets reproduces NEXT_AGENT gap #3: an MCP whose
// targets name a tool that is not claude/opencode matches no adapter and is
// silently projected nowhere. Load must fail naming the unknown target.
func TestLoadRejectsUnknownTargets(t *testing.T) {
	bad := []struct{ label, doc, offender string }{
		{"typo", "[mcps.x]\ncommand=[\"c\"]\ntargets=[\"claud\"]\n", "claud"},
		{"unknown tool", "[mcps.x]\ncommand=[\"c\"]\ntargets=[\"vscode\"]\n", "vscode"},
		{"one good one bad", "[mcps.x]\ncommand=[\"c\"]\ntargets=[\"claude\",\"opencde\"]\n", "opencde"},
	}
	for _, tc := range bad {
		err := loadDoc(t, tc.doc)
		if err == nil {
			t.Fatalf("%s: unknown target %q accepted; want load error", tc.label, tc.offender)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.offender)) {
			t.Fatalf("%s: error does not name the offender %q: %v", tc.label, tc.offender, err)
		}
	}
	if err := loadDoc(t, "[mcps.x]\ncommand=[\"c\"]\ntargets=[\"claude\",\"opencode\"]\n"); err != nil {
		t.Fatalf("valid targets rejected: %v", err)
	}
	// No targets means all tools — still valid.
	if err := loadDoc(t, "[mcps.x]\ncommand=[\"c\"]\n"); err != nil {
		t.Fatalf("default targets rejected: %v", err)
	}
}

// TestLoadRejectsEmptyCommand reproduces gap #3: an MCP with no runnable
// command is skipped by both adapters (desired() len(Command)==0), a silent
// no-op. Load must fail naming the MCP that cannot project.
func TestLoadRejectsEmptyCommand(t *testing.T) {
	for _, tc := range []struct{ label, doc string }{
		{"missing command", "[mcps.foo]\ntargets=[\"claude\"]\n"},
		{"empty command", "[mcps.foo]\ncommand=[]\n"},
	} {
		err := loadDoc(t, tc.doc)
		if err == nil {
			t.Fatalf("%s: accepted; want load error", tc.label)
		}
		if !strings.Contains(err.Error(), strconv.Quote("foo")) {
			t.Fatalf("%s: error does not name the MCP %q: %v", tc.label, "foo", err)
		}
	}
}

// TestLoadRejectsReservedSettingKeys reproduces gap #3: a settings key that
// collides with a structure homonto itself manages in the same tool file
// (claude enabledPlugins in settings.json; opencode mcp/plugin in
// opencode.jsonc) must be a load error, not a silent fight at apply.
func TestLoadRejectsReservedSettingKeys(t *testing.T) {
	for _, tc := range []struct{ label, doc, key string }{
		{"claude enabledPlugins", "[settings.claude]\nenabledPlugins={}\n", "enabledPlugins"},
		{"opencode mcp", "[settings.opencode]\nmcp={}\n", "mcp"},
		{"opencode plugin", "[settings.opencode]\nplugin=[]\n", "plugin"},
	} {
		err := loadDoc(t, tc.doc)
		if err == nil {
			t.Fatalf("%s: reserved key accepted; want load error", tc.label)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.key)) {
			t.Fatalf("%s: error does not name the key %q: %v", tc.label, tc.key, err)
		}
	}
	// Exact collisions only: the same names are fine in the OTHER tool, and
	// non-colliding keys load normally. (Note: settings.claude.mcpServers also
	// loads, but is deliberately NOT asserted here — claude's current() skips
	// reading it back from settings.json, so it is non-idempotent at apply; it
	// is a separate pre-existing adapter concern, not a "harmless" key.)
	for _, ok := range []string{
		"[settings.claude]\nmcp={}\n",              // read back by current(); idempotent
		"[settings.opencode]\nenabledPlugins={}\n", // reserved for claude only, fine for opencode
		"[settings.claude]\nmodel=\"opus\"\n",
	} {
		if err := loadDoc(t, ok); err != nil {
			t.Fatalf("non-reserved settings rejected: %v (doc %q)", err, ok)
		}
	}
}
