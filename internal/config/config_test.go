package config

import (
	"os"
	"path/filepath"
	"reflect"
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

[frameworks.onto]
source = "builtin:onto"
scope = "project"

[skills.graphify]
source = "local:graphify"
scope = "project"

[skills.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[commands.review]
source = "builtin:review"
scope = "project"
targets = ["opencode"]

[subagents.architect]
source = "builtin:architect"
scope = "project"

[plugins.claude.claude-hud]
source = "claude-hud@official"
enabled = true

[plugins.opencode.quota]
source = "@slkiser/opencode-quota"

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"

[models.claude.architectural]
model = "opus"
variant = "max"

[models.claude.coding]
model = "sonnet"
effort = "normal"

[models.claude.trivial]
model = "haiku"
effort = "fast"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"

[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"

[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
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
	if got := c.Frameworks["onto"].Scope; got != "project" {
		t.Fatalf("framework onto scope = %q", got)
	}
	if got := c.Skills["graphify"].Source; got != "local:graphify" {
		t.Fatalf("skill graphify source = %q", got)
	}
	claudeSkills := c.SkillEntriesForTool("claude")
	if len(claudeSkills) != 2 || claudeSkills[0].Name != "comet" || claudeSkills[1].Name != "graphify" {
		t.Fatalf("claude skill entries = %#v", claudeSkills)
	}
	opencodeSkills := c.SkillEntriesForTool("opencode")
	if len(opencodeSkills) != 1 || opencodeSkills[0].Name != "graphify" {
		t.Fatalf("opencode skill entries = %#v", opencodeSkills)
	}
	if got := c.Models.Claude["architectural"].Variant; got != "max" {
		t.Fatalf("claude architectural variant = %q", got)
	}
	// Plugin declaration tables parse into per-tool maps keyed by decl name,
	// carrying source and (default-true) enabled.
	if got := c.Plugins.Claude["claude-hud"]; got.Source != "claude-hud@official" || !got.IsEnabled() {
		t.Fatalf("claude plugin claude-hud = %#v", got)
	}
	oc := c.Plugins.OpenCode["quota"]
	if oc.Source != "@slkiser/opencode-quota" || !oc.IsEnabled() {
		t.Fatalf("opencode plugin quota = %#v (enabled default should be true)", oc)
	}
}

// TestLoadPluginEnabledSemantics covers the enabled flag: omitted defaults to
// true (enabled), false disables.
func TestLoadPluginEnabledSemantics(t *testing.T) {
	doc := "[plugins.claude.on]\nsource = \"on@m\"\n" +
		"[plugins.claude.off]\nsource = \"off@m\"\nenabled = false\n"
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !c.Plugins.Claude["on"].IsEnabled() {
		t.Fatalf("plugin with omitted enabled should default enabled")
	}
	if c.Plugins.Claude["off"].IsEnabled() {
		t.Fatalf("plugin with enabled=false should be disabled")
	}
}

// TestLoadPluginConfig: a claude plugin may carry a [plugins.claude.<name>.config]
// table of non-sensitive options, parsed into Plugin.Config.
func TestLoadPluginConfig(t *testing.T) {
	doc := "[plugins.claude.hud]\nsource = \"hud@official\"\n" +
		"[plugins.claude.hud.config]\napi_endpoint = \"https://x\"\nmax_workers = 4\n"
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg := c.Plugins.Claude["hud"].Config
	if cfg["api_endpoint"] != "https://x" {
		t.Fatalf("plugin config api_endpoint = %#v; want \"https://x\"", cfg["api_endpoint"])
	}
	if got, ok := cfg["max_workers"].(int64); !ok || got != 4 {
		t.Fatalf("plugin config max_workers = %#v; want int64(4)", cfg["max_workers"])
	}
}

// TestLoadRejectsOpenCodePluginConfig: OpenCode has no per-plugin config on disk
// (its plugins are a plain array), so a [plugins.opencode.<name>.config] table
// cannot project. Load must fail naming the plugin.
func TestLoadRejectsOpenCodePluginConfig(t *testing.T) {
	doc := "[plugins.opencode.q]\nsource = \"q\"\n" +
		"[plugins.opencode.q.config]\nfoo = \"bar\"\n"
	err := loadDoc(t, doc)
	if err == nil {
		t.Fatal("opencode plugin config accepted; want load error")
	}
	if !strings.Contains(err.Error(), strconv.Quote("q")) {
		t.Fatalf("error does not name the plugin %q: %v", "q", err)
	}
}

// TestLoadRejectsEmptyPluginSource: a plugin declaration whose source is empty
// (or whitespace) cannot project anywhere, so Load must fail naming the plugin.
func TestLoadRejectsEmptyPluginSource(t *testing.T) {
	for _, tc := range []struct{ label, doc, name string }{
		{"claude missing source", "[plugins.claude.hud]\n", "hud"},
		{"claude empty source", "[plugins.claude.hud]\nsource = \"\"\n", "hud"},
		{"opencode whitespace source", "[plugins.opencode.q]\nsource = \"   \"\n", "q"},
	} {
		err := loadDoc(t, tc.doc)
		if err == nil {
			t.Fatalf("%s: empty source accepted; want load error", tc.label)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.name)) {
			t.Fatalf("%s: error does not name the plugin %q: %v", tc.label, tc.name, err)
		}
	}
}

// TestLoadRejectsDuplicatePluginSource: two decl names sharing one source would
// collide on the single projected key (keyed by source), giving a
// last-writer-wins, iteration-order-dependent plan. Load must reject it.
func TestLoadRejectsDuplicatePluginSource(t *testing.T) {
	doc := "[plugins.claude.hud]\nsource = \"hud@official\"\n" +
		"[plugins.claude.hud-off]\nsource = \"hud@official\"\nenabled = false\n"
	err := loadDoc(t, doc)
	if err == nil {
		t.Fatal("duplicate source accepted; want load error")
	}
	if !strings.Contains(err.Error(), "hud@official") {
		t.Fatalf("error does not name the shared source: %v", err)
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
		{"claude plugin", "[plugins.claude.\"7\"]\nsource = \"x\"\n", "7"},
		{"opencode plugin", "[plugins.opencode.\"\"]\nsource = \"x\"\n", ""},
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

// TestLoadParsesTUIOpenCode: a [tui.opencode] table parses into TUI.OpenCode as
// a free-form map, mirroring [settings.opencode]. These keys project to a
// second managed file (~/.config/opencode/tui.json).
func TestLoadParsesTUIOpenCode(t *testing.T) {
	doc := "[tui.opencode]\ntheme = \"gruvbox\"\nscroll_speed = 3\n"
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := c.TUI.OpenCode["theme"]; got != "gruvbox" {
		t.Fatalf("tui.opencode theme = %#v; want \"gruvbox\"", got)
	}
	if got, ok := c.TUI.OpenCode["scroll_speed"].(int64); !ok || got != 3 {
		t.Fatalf("tui.opencode scroll_speed = %#v; want int64(3)", c.TUI.OpenCode["scroll_speed"])
	}
}

// TestLoadRejectsTUIIndexLikeName: like [settings.opencode], a [tui.opencode]
// key that sjson would treat as an array index ("0", "-1") or an empty key
// would corrupt tui.json. Load must reject it naming the offending entry.
func TestLoadRejectsTUIIndexLikeName(t *testing.T) {
	for _, tc := range []struct{ label, doc, name string }{
		{"tui zero", "[tui.opencode]\n\"0\" = \"x\"\n", "0"},
		{"tui minus-one", "[tui.opencode]\n\"-1\" = \"x\"\n", "-1"},
		{"tui empty", "[tui.opencode]\n\"\" = \"x\"\n", ""},
	} {
		err := loadDoc(t, tc.doc)
		if err == nil {
			t.Fatalf("%s: name %q accepted; want load error", tc.label, tc.name)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.name)) {
			t.Fatalf("%s: error does not name the entry %q: %v", tc.label, tc.name, err)
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

// TestSubagentModeValidation: subagents accept link (default) and copy; an
// unknown mode is invalid.
func TestSubagentModeValidation(t *testing.T) {
	base := func(mode string) string {
		m := ""
		if mode != "" {
			m = "mode=\"" + mode + "\"\n"
		}
		return "[subagents.x]\nsource=\"builtin:architect\"\nscope=\"user\"\n" + m + validModelsBothTools()
	}
	for _, mode := range []string{"", "link", "copy"} {
		if err := loadDoc(t, base(mode)); err != nil {
			t.Fatalf("mode %q must load: %v", mode, err)
		}
	}
	if err := loadDoc(t, base("bogus")); err == nil {
		t.Fatal("an unknown subagent mode must be rejected")
	}
}

// TestAgentSupersededIntoSubagent: an [agents.<name>] is folded into an
// equivalent copy-mode [subagents.<name>] at load (Option C), the agents table is
// cleared, and a declared agent supersedes an explicit same-name subagent.
func TestAgentSupersededIntoSubagent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "homonto.toml")
	load := func(doc string) *Config {
		if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		c, err := Load(p)
		if err != nil {
			t.Fatalf("load: %v\n%s", err, doc)
		}
		return c
	}

	// A builtin agent supersedes to a COPY-mode subagent (builtin was copy-only).
	c := load("[agents.rev]\nsource=\"builtin:code-reviewer\"\ntargets=[\"claude\"]\n" +
		"[models.claude.architectural]\nmodel=\"o\"\nvariant=\"m\"\n[models.claude.coding]\nmodel=\"o\"\nvariant=\"m\"\n[models.claude.trivial]\nmodel=\"o\"\nvariant=\"m\"\n")
	if len(c.Agents) != 0 {
		t.Fatal("the [agents] table must be cleared after supersede")
	}
	sa, ok := c.Subagents["rev"]
	if !ok {
		t.Fatal("agent rev was not superseded into a subagent")
	}
	if sa.Mode != "copy" {
		t.Fatalf("a builtin agent must supersede to copy mode, got %q", sa.Mode)
	}
	if sa.Scope != "user" {
		t.Fatalf("a superseded agent must keep user scope, got %q", sa.Scope)
	}

	// A declared [agents.X] wins over an explicit [subagents.X] of the same name.
	c2 := load("[agents.dup]\nsource=\"local:dup\"\nmode=\"copy\"\ntargets=[\"claude\"]\n" +
		"[subagents.dup]\nsource=\"builtin:architect\"\nscope=\"project\"\ntargets=[\"claude\"]\n" +
		"[models.claude.architectural]\nmodel=\"o\"\nvariant=\"m\"\n[models.claude.coding]\nmodel=\"o\"\nvariant=\"m\"\n[models.claude.trivial]\nmodel=\"o\"\nvariant=\"m\"\n")
	if got := c2.Subagents["dup"].Source; got != "local:dup" {
		t.Fatalf("the agent declaration must win the name; subagent source = %q", got)
	}
}

// TestSubagentScopeDefaultsToProject: an omitted [subagents.<name>] scope is no
// longer an error — it defaults to project (Option C step 1). An explicit scope
// is still honored, and skills/commands still require scope.
func TestSubagentScopeDefaultsToProject(t *testing.T) {
	p := filepath.Join(t.TempDir(), "homonto.toml")
	load := func(doc string) (*Config, error) {
		if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		return Load(p)
	}

	c, err := load("[subagents.architect]\nsource=\"builtin:architect\"\n" + validModelsBothTools())
	if err != nil {
		t.Fatalf("omitted subagent scope should default to project, not error: %v", err)
	}
	if got := c.Subagents["architect"].Scope; got != "project" {
		t.Fatalf("omitted subagent scope = %q, want \"project\"", got)
	}

	c2, err := load("[subagents.architect]\nsource=\"builtin:architect\"\nscope=\"user\"\n" + validModelsBothTools())
	if err != nil {
		t.Fatalf("explicit subagent scope: %v", err)
	}
	if got := c2.Subagents["architect"].Scope; got != "user" {
		t.Fatalf("explicit subagent scope = %q, want \"user\"", got)
	}

	// Skills still require an explicit scope.
	if err := loadDoc(t, "[skills.s]\nsource=\"local:s\"\n"); err == nil {
		t.Fatal("a skill with no scope must still be rejected")
	}
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
		{"claude mcpServers", "[settings.claude]\nmcpServers={}\n", "mcpServers"},
		{"claude pluginConfigs", "[settings.claude]\npluginConfigs={}\n", "pluginConfigs"},
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
	// non-colliding keys load normally. (settings.claude.mcpServers is now
	// rejected above — claude's current() skips reading it back from
	// settings.json, so it would be non-idempotent at apply.)
	for _, ok := range []string{
		"[settings.claude]\nmcp={}\n",              // read back by current(); idempotent
		"[settings.opencode]\nenabledPlugins={}\n", // reserved for claude only, fine for opencode
		"[settings.opencode]\nmcpServers={}\n",     // reserved for claude only, fine for opencode
		"[settings.claude]\nmodel=\"opus\"\n",
	} {
		if err := loadDoc(t, ok); err != nil {
			t.Fatalf("non-reserved settings rejected: %v (doc %q)", err, ok)
		}
	}
}

func TestLoadRejectsBadResourceNames(t *testing.T) {
	for _, tc := range []struct{ kind, table, name string }{
		{"framework", "frameworks", "../evil"},
		{"skill", "skills", ".."},
		{"command", "commands", ""},
		{"subagent", "subagents", "a/b"},
		{"subagent", "subagents", `a\b`},
		{"skill", "skills", "0"},
	} {
		doc := "[" + tc.table + "." + strconv.Quote(tc.name) + "]\nsource=\"local:x\"\nscope=\"project\"\n" + validModelsBothTools()
		err := loadDoc(t, doc)
		if err == nil {
			t.Fatalf("%s name %q accepted; want load error", tc.kind, tc.name)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.name)) {
			t.Fatalf("error for %q does not name the entry: %v", tc.name, err)
		}
	}
}

func TestLoadRejectsResourceWithoutExplicitScope(t *testing.T) {
	err := loadDoc(t, "[skills.graphify]\nsource=\"local:graphify\"\n"+validModelsBothTools())
	if err == nil {
		t.Fatal("resource without scope accepted; want load error")
	}
	for _, want := range []string{"skills.graphify", "scope"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %v does not mention %q", err, want)
		}
	}
}

func TestLoadRejectsInvalidResourceScope(t *testing.T) {
	err := loadDoc(t, "[commands.review]\nsource=\"builtin:review\"\nscope=\"global\"\n"+validModelsBothTools())
	if err == nil {
		t.Fatal("scope global accepted; want load error")
	}
	for _, want := range []string{`"global"`, "user", "project"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %v does not mention %q", err, want)
		}
	}
}

func TestLoadRejectsInvalidResourceSource(t *testing.T) {
	for _, source := range []string{"", "https://example.com/x", "github:owner/repo", "builtin:", "local:"} {
		doc := "[skills.graphify]\nsource=" + strconv.Quote(source) + "\nscope=\"project\"\n" + validModelsBothTools()
		err := loadDoc(t, doc)
		if err == nil {
			t.Fatalf("source %q accepted; want load error", source)
		}
		if !strings.Contains(err.Error(), strconv.Quote(source)) {
			t.Fatalf("error %v does not name source %q", source, err)
		}
	}
}

func TestLoadRejectsUnknownResourceTargets(t *testing.T) {
	err := loadDoc(t, "[subagents.architect]\nsource=\"builtin:architect\"\nscope=\"project\"\ntargets=[\"claud\"]\n"+validModelsBothTools())
	if err == nil {
		t.Fatal("unknown target accepted; want load error")
	}
	if !strings.Contains(err.Error(), strconv.Quote("claud")) {
		t.Fatalf("error does not name unknown target: %v", err)
	}
}

func TestLoadRequiresAllModelLevelsForEnabledTools(t *testing.T) {
	doc := `
[commands.review]
source = "builtin:review"
scope = "project"
targets = ["opencode"]

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"

[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
`
	err := loadDoc(t, doc)
	if err == nil {
		t.Fatal("missing opencode trivial model accepted; want load error")
	}
	for _, want := range []string{"models.opencode.trivial", "model"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %v does not mention %q", err, want)
		}
	}
}

func TestLoadRequiresModelAndEffortOrVariant(t *testing.T) {
	doc := `
[commands.review]
source = "builtin:review"
scope = "project"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"

[models.claude.coding]
model = "sonnet"

[models.claude.trivial]
model = "haiku"
effort = "fast"
`
	err := loadDoc(t, doc)
	if err == nil {
		t.Fatal("model without effort or variant accepted; want load error")
	}
	if !strings.Contains(err.Error(), "models.claude.coding") {
		t.Fatalf("error does not name route: %v", err)
	}
}

func TestLoadDoesNotRequireModelsForSkillsOnly(t *testing.T) {
	err := loadDoc(t, "[skills.graphify]\nsource=\"local:graphify\"\nscope=\"project\"\n")
	if err != nil {
		t.Fatalf("skills-only config required model routing: %v", err)
	}
}

// TestEnabledModelTools locks the rule that model routing is derived only from
// frameworks/commands/subagents — [skills.*] never counts, because skills-only
// configs do not need models. Returns the sorted union of targeted tools.
func TestEnabledModelTools(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  *Config
		want []string
	}{
		{
			name: "skills only does not enable any model tool",
			cfg: &Config{Skills: map[string]Resource{
				"x": {Source: "local:x", Scope: "user"},
			}},
			want: []string{},
		},
		{
			name: "single command with one target",
			cfg: &Config{Commands: map[string]Resource{
				"x": {Source: "builtin:x", Scope: "project", Targets: []string{"claude"}},
			}},
			want: []string{"claude"},
		},
		{
			name: "framework with no targets defaults to both",
			cfg: &Config{Frameworks: map[string]Resource{
				"x": {Source: "builtin:x", Scope: "project"},
			}},
			want: []string{"claude", "opencode"},
		},
		{
			name: "mixed frameworks+subagents+skills union (skills ignored)",
			cfg: &Config{
				Frameworks: map[string]Resource{
					"a": {Source: "builtin:a", Scope: "project"},
				},
				Subagents: map[string]Subagent{
					"b": {Source: "builtin:b", Scope: "project", Targets: []string{"opencode"}},
				},
				Skills: map[string]Resource{
					"c": {Source: "local:c", Scope: "user"},
				},
			},
			want: []string{"claude", "opencode"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.EnabledModelTools()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("EnabledModelTools = %v, want %v", got, tc.want)
			}
		})
	}
}

func loadTOML(t *testing.T, body string) *Config {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "homonto.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return c
}

func TestExpandedSkillsIncludeFrameworkAndDeps(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`)
	got, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	byName := map[string]NamedResource{}
	for _, e := range got {
		byName[e.Name] = e
	}
	// A comet skill, a superpowers dep skill, and an openspec dep skill.
	for _, want := range []string{"comet-open", "brainstorming", "openspec-explore"} {
		e, ok := byName[want]
		if !ok {
			t.Fatalf("expanded set missing %q; got %v", want, keysOf(byName))
		}
		if e.Resource.Source != "builtin:"+want {
			t.Fatalf("%q source = %q", want, e.Resource.Source)
		}
		// Inherits the framework declaration's scope and targets (Spec Patch #1).
		if e.Resource.Scope != "user" || len(e.Resource.Targets) != 1 || e.Resource.Targets[0] != "claude" {
			t.Fatalf("%q did not inherit scope/targets: %+v", want, e.Resource)
		}
	}
}

func keysOf(m map[string]NamedResource) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func TestExpandedSkillsTargetFiltering(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`)
	got, err := c.ExpandedSkillEntriesForTool("opencode")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("comet targets claude only; opencode should get no skills, got %v", got)
	}
}

func TestExpandedSkillsCollisionWithExplicit(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[skills.comet-open]
source = "builtin:comet-open"
scope = "user"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`)
	_, err := c.ExpandedSkillEntriesForTool("claude")
	if err == nil || !strings.Contains(err.Error(), "comet-open") {
		t.Fatalf("expected collision error naming comet-open, got %v", err)
	}
}

// TestExpandedSkillsFrameworkVsFrameworkConflict reproduces the reviewer's
// framework-vs-framework collision path: two frameworks both expand
// "comet-open" (and the rest of the comet catalog) via the REAL embedded
// catalog, but with different scope, so the second framework's declaration
// conflicts with the first's. ExpandedSkillEntriesForTool must error.
func TestExpandedSkillsFrameworkVsFrameworkConflict(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet_a]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[frameworks.comet_b]
source = "builtin:comet"
scope = "project"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`)
	_, err := c.ExpandedSkillEntriesForTool("claude")
	if err == nil {
		t.Fatal("expected conflict error for two frameworks expanding the same skill with different scope, got nil")
	}
	if !strings.Contains(err.Error(), "comet-open") && !strings.Contains(err.Error(), "comet_b") {
		t.Fatalf("error does not name the conflicting skill or framework: %v", err)
	}
}

// TestExpandedSkillsSameFrameworkDeclDedup reproduces the reviewer's
// same-skill-same-declaration dedup path: two frameworks both expand
// "comet-open" via the REAL embedded catalog, with IDENTICAL scope and
// targets, so they should collapse into one entry with no error.
func TestExpandedSkillsSameFrameworkDeclDedup(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet_a]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[frameworks.comet_b]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`)
	got, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	count := 0
	for _, e := range got {
		if e.Name == "comet-open" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("comet-open should appear exactly once (deduped), got %d occurrences in %v", count, got)
	}
}

func validModelsBothTools() string {
	return `
[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "normal"
[models.claude.trivial]
model = "haiku"
effort = "fast"
[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
`
}

func TestExpandedCommandsExplicitAndTargetFilter(t *testing.T) {
	c := loadTOML(t, `
[commands.example-command]
source = "builtin:example-command"
scope = "project"
targets = ["claude"]
`+validModelsBothTools())

	claude, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand claude: %v", err)
	}
	if len(claude) != 1 || claude[0].Name != "example-command" {
		t.Fatalf("claude commands = %v", claude)
	}
	if claude[0].Resource.Source != "builtin:example-command" || claude[0].Resource.Scope != "project" {
		t.Fatalf("example-command resource = %+v", claude[0].Resource)
	}
	// targets = ["claude"] only -> opencode gets nothing.
	opencode, err := c.ExpandedCommandEntriesForTool("opencode")
	if err != nil {
		t.Fatalf("expand opencode: %v", err)
	}
	if len(opencode) != 0 {
		t.Fatalf("opencode commands = %v, want none", opencode)
	}
}

// A skill and a command may share a name: separate namespaces, both returned.
func TestSkillAndCommandMayShareName(t *testing.T) {
	c := loadTOML(t, `
[skills.shared]
source = "builtin:shared"
scope = "user"
targets = ["claude"]

[commands.shared]
source = "builtin:shared"
scope = "user"
targets = ["claude"]
`+validModelsBothTools())

	skills, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	commands, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		t.Fatalf("commands: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != "shared" {
		t.Fatalf("skills = %v", skills)
	}
	if len(commands) != 1 || commands[0].Name != "shared" {
		t.Fatalf("commands = %v", commands)
	}
}

// The framework loop must not crash or invent commands when the real framework
// declares no [commands] table: only explicit commands survive.
func TestExpandedCommandsFrameworkWithoutCommandsNoOps(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[commands.example-command]
source = "builtin:example-command"
scope = "user"
targets = ["claude"]
`+validModelsBothTools())

	got, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(got) != 1 || got[0].Name != "example-command" {
		t.Fatalf("commands = %v, want only example-command", got)
	}
}

func TestExpandedSubagentsExplicitAndTargetFilter(t *testing.T) {
	c := loadTOML(t, `
[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"
targets = ["claude"]
`+validModelsBothTools())

	claude, err := c.ExpandedSubagentEntriesForTool("claude")
	if err != nil {
		t.Fatalf("claude: %v", err)
	}
	if len(claude) != 1 || claude[0].Name != "code-reviewer" {
		t.Fatalf("claude subagents = %+v, want [code-reviewer]", claude)
	}
	oc, err := c.ExpandedSubagentEntriesForTool("opencode")
	if err != nil {
		t.Fatalf("opencode: %v", err)
	}
	if len(oc) != 0 {
		t.Fatalf("opencode subagents = %+v, want none (target filter)", oc)
	}
}

func TestExpandedSubagentsFrameworkInheritsScopeTargets(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "project"
targets = ["claude", "opencode"]
`+validModelsBothTools())

	got, err := c.ExpandedSubagentEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	var nav *NamedResource
	for i := range got {
		if got[i].Name == "comet-navigator" {
			nav = &got[i]
		}
	}
	if nav == nil {
		t.Fatal("comet-navigator not expanded for claude")
	}
	if nav.Resource.Scope != "project" || nav.Resource.Source != "builtin:comet-navigator" {
		t.Fatalf("comet-navigator inherited wrong scope/source: %+v", nav.Resource)
	}
}

func TestExpandedSubagentsExplicitVsFrameworkCollision(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "project"

[subagents.comet-navigator]
source = "builtin:comet-navigator"
scope = "user"
`+validModelsBothTools())

	if _, err := c.ExpandedSubagentEntriesForTool("claude"); err == nil {
		t.Fatal("expected collision error: comet-navigator declared explicitly and by framework")
	}
}

// EnabledModelTools already iterates c.Subagents, so a subagent targeting a
// tool with no model routes must already fail at Load. This test locks that
// behavior in place — no production change should be needed for it to pass.
func TestLoadRequiresModelsForSubagentTargetedTool(t *testing.T) {
	doc := `
[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"
targets = ["opencode"]
`
	if err := loadDoc(t, doc); err == nil {
		t.Fatal("subagent enabling opencode without model routes was accepted; want load error")
	} else if !strings.Contains(err.Error(), "models.opencode") {
		t.Fatalf("error %v does not mention missing opencode model routes", err)
	}
}

// TestLoadMarketplace: a [marketplaces.claude.<name>] github declaration parses
// into Marketplaces.Claude with source, repo, and optional auto_update.
func TestLoadMarketplace(t *testing.T) {
	doc := "[marketplaces.claude.official]\nsource = \"github\"\nrepo = \"anthropics/claude-plugins\"\nauto_update = true\n"
	p := filepath.Join(t.TempDir(), "homonto.toml")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	mk := c.Marketplaces.Claude["official"]
	if mk.Source != "github" {
		t.Fatalf("marketplace source = %q; want github", mk.Source)
	}
	if mk.Repo != "anthropics/claude-plugins" {
		t.Fatalf("marketplace repo = %q; want anthropics/claude-plugins", mk.Repo)
	}
	if mk.AutoUpdate == nil || *mk.AutoUpdate != true {
		t.Fatalf("marketplace auto_update = %#v; want *true", mk.AutoUpdate)
	}
}

// TestLoadRejectsUnknownMarketplaceSource: a marketplace whose source is not one
// of the four known kinds cannot project. Load must fail naming the marketplace
// and the bad source.
func TestLoadRejectsUnknownMarketplaceSource(t *testing.T) {
	doc := "[marketplaces.claude.weird]\nsource = \"svn\"\nrepo = \"x/y\"\n"
	err := loadDoc(t, doc)
	if err == nil {
		t.Fatal("unknown marketplace source accepted; want load error")
	}
	if !strings.Contains(err.Error(), strconv.Quote("weird")) {
		t.Fatalf("error does not name the marketplace %q: %v", "weird", err)
	}
	if !strings.Contains(err.Error(), strconv.Quote("svn")) {
		t.Fatalf("error does not name the source %q: %v", "svn", err)
	}
}

// TestLoadRejectsMarketplaceMissingLocator: each source kind requires its
// locator field(s); a github marketplace without repo, or a git-subdir without
// url/path, must be a load error naming the marketplace.
func TestLoadRejectsMarketplaceMissingLocator(t *testing.T) {
	for _, tc := range []struct{ label, doc, name string }{
		{"github no repo", "[marketplaces.claude.official]\nsource = \"github\"\n", "official"},
		{"git-subdir no url", "[marketplaces.claude.sub]\nsource = \"git-subdir\"\npath = \"p\"\n", "sub"},
		{"git-subdir no path", "[marketplaces.claude.sub]\nsource = \"git-subdir\"\nurl = \"https://x\"\n", "sub"},
		{"url no url", "[marketplaces.claude.u]\nsource = \"url\"\n", "u"},
		{"directory no path", "[marketplaces.claude.d]\nsource = \"directory\"\n", "d"},
	} {
		err := loadDoc(t, tc.doc)
		if err == nil {
			t.Fatalf("%s: missing locator accepted; want load error", tc.label)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.name)) {
			t.Fatalf("%s: error does not name the marketplace %q: %v", tc.label, tc.name, err)
		}
	}
}

// TestLoadRejectsReservedMarketplaceSetting: settings.claude.extraKnownMarketplaces
// collides with homonto's own marketplace projection into settings.json, so it
// must be a load error like the other reserved settings keys.
func TestLoadRejectsReservedMarketplaceSetting(t *testing.T) {
	err := loadDoc(t, "[settings.claude]\nextraKnownMarketplaces={}\n")
	if err == nil {
		t.Fatal("reserved key extraKnownMarketplaces accepted; want load error")
	}
	if !strings.Contains(err.Error(), strconv.Quote("extraKnownMarketplaces")) {
		t.Fatalf("error does not name the key: %v", err)
	}
}

// TestAgentsParseFullDeclaration: a fully specified [agents.<name>] parses into
// c.Agents with source/version/targets/mode preserved verbatim.

// TestAgentsRejectInvalidSource: a non builtin:/local: source is rejected,
// naming the agent.
func TestAgentsRejectInvalidSource(t *testing.T) {
	err := loadDoc(t, "[agents.review]\nsource = \"https://example.com/x\"\n")
	if err == nil {
		t.Fatalf("invalid source accepted; want load error")
	}
	if !strings.Contains(err.Error(), "review") || !strings.Contains(err.Error(), "source") {
		t.Fatalf("error does not name the agent+source: %v", err)
	}
}

// TestAgentsRejectTraversalName: an agent name with path components is rejected
// (agents are projected to files named by the agent name in later increments, so
// a "../x" name must not survive declaration).
func TestAgentsRejectTraversalName(t *testing.T) {
	err := loadDoc(t, "[agents.\"../evil\"]\nsource = \"builtin:x\"\n")
	if err == nil {
		t.Fatalf("traversal agent name accepted; want load error")
	}
	if !strings.Contains(err.Error(), "not a plain name") {
		t.Fatalf("error does not flag the bad name: %v", err)
	}
}

// TestAgentsRejectTraversalLocalSource: a local: source with path components is
// rejected — it would resolve/materialize a file outside homonto/agents/ on
// `agents add` (a config-driven path-traversal / file-exfiltration vector).
func TestAgentsRejectTraversalLocalSource(t *testing.T) {
	err := loadDoc(t, "[agents.rev]\nsource = \"local:../../secret\"\n")
	if err == nil {
		t.Fatalf("traversal local source accepted; want load error")
	}
	if !strings.Contains(err.Error(), "plain name") {
		t.Fatalf("error does not flag the bad source: %v", err)
	}
}

// TestResourcesRejectTraversalLocalSource: a local: source with path components
// is rejected for skills AND commands (F28) — the same plain-name rule subagents
// already enforce, so a local:../../x can never join a traversal suffix into a
// provider path.
func TestResourcesRejectTraversalLocalSource(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{"skill", "[skills.x]\nsource = \"local:../../etc/x\"\nscope = \"user\"\n"},
		{"command", "[commands.y]\nsource = \"local:../y\"\nscope = \"user\"\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := loadDoc(t, tc.doc)
			if err == nil {
				t.Fatalf("traversal local %s source accepted; want load error", tc.name)
			}
			if !strings.Contains(err.Error(), "plain name") {
				t.Fatalf("error does not flag the bad source: %v", err)
			}
		})
	}
}

// TestResourcesAcceptPlainLocalSource: a plain local: name passes for skills and
// commands (the non-traversal counterpart to the rejection above).
func TestResourcesAcceptPlainLocalSource(t *testing.T) {
	docs := []string{
		"[skills.x]\nsource = \"local:x\"\nscope = \"user\"\n" + validModelsBothTools(),
		"[commands.y]\nsource = \"local:y\"\nscope = \"user\"\n" + validModelsBothTools(),
	}
	for _, doc := range docs {
		if err := loadDoc(t, doc); err != nil {
			t.Fatalf("plain local source must load, got %v for:\n%s", err, doc)
		}
	}
}

// TestAgentsRejectInvalidMode: a mode outside copy/link is rejected, naming the
// agent and the offending mode.
func TestAgentsRejectInvalidMode(t *testing.T) {
	err := loadDoc(t, "[agents.review]\nsource = \"builtin:x\"\nmode = \"symlink\"\n")
	if err == nil {
		t.Fatalf("invalid mode accepted; want load error")
	}
	if !strings.Contains(err.Error(), "review") || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error does not name the agent+mode: %v", err)
	}
}

// TestAgentsRejectUnknownTarget: a target naming a tool other than
// claude/opencode is rejected, naming the offending target.
func TestAgentsRejectUnknownTarget(t *testing.T) {
	err := loadDoc(t, "[agents.review]\nsource = \"builtin:x\"\ntargets = [\"vscode\"]\n")
	if err == nil {
		t.Fatalf("unknown target accepted; want load error")
	}
	if !strings.Contains(err.Error(), "vscode") {
		t.Fatalf("error does not name the unknown target: %v", err)
	}
}
