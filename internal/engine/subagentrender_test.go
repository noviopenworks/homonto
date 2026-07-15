package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// opencodeSubagentTOML installs a builtin subagent for OpenCode, whose rendered
// frontmatter is stamped with the [models.opencode.*] routes. %s is the
// architectural model — the route the onto-reviewer's `role: architectural`
// resolves through.
const opencodeSubagentTOML = `
[subagents.onto-reviewer]
source = "builtin:onto-reviewer"
scope = "project"
targets = ["opencode"]

[models.opencode.architectural]
model = "%s"
variant = "high"
[models.opencode.coding]
model = "some/coding-model"
effort = "medium"
[models.opencode.trivial]
model = "some/trivial-model"
effort = "medium"
`

func writeConfig(t *testing.T, repo, model string) {
	t.Helper()
	body := strings.Replace(opencodeSubagentTOML, "%s", model, 1)
	if err := os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func renderedModel(t *testing.T, e *Engine, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(e.SubagentDir(), file))
	if err != nil {
		t.Fatalf("read rendered subagent %s: %v", file, err)
	}
	for _, ln := range strings.Split(string(data), "\n") {
		if m, ok := strings.CutPrefix(ln, "model: "); ok {
			return m
		}
	}
	return ""
}

// TestApplyRerendersSubagentsWhenModelRouteChanges is the regression guard for
// the stale-render bug: materializeCatalog was gated on the catalog version and
// file existence alone, but a subagent's rendered `model:` comes from the
// config's model routes. Editing a route left the catalog version untouched, so
// the gate short-circuited and the projected agent kept its OLD model forever —
// while the tool's own setting.model (re-read from the routes each apply)
// correctly moved. Same config, two different answers.
func TestApplyRerendersSubagentsWhenModelRouteChanges(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	writeConfig(t, repo, "first/model-a")
	e := buildEngine(t, home, repo)
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if got := renderedModel(t, e, "onto-reviewer.opencode.md"); got != "first/model-a" {
		t.Fatalf("after first apply: rendered model = %q, want %q", got, "first/model-a")
	}

	// Change ONLY the architectural route. The catalog is byte-for-byte identical.
	writeConfig(t, repo, "second/model-b")
	e2 := buildEngine(t, home, repo)
	if err := e2.Apply(mustPlan(t, e2)); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if got := renderedModel(t, e2, "onto-reviewer.opencode.md"); got != "second/model-b" {
		t.Fatalf("after route change: rendered model = %q, want %q (agent frozen at the old route)", got, "second/model-b")
	}
}

// TestApplyRestoresDeletedRenderedVariant guards the other half of the same
// gate: allSubagentFilesExist checked only the shared <name>.md anchor, never
// the per-tool <name>.<tool>.md variant the adapter actually links. A deleted
// variant left the anchor in place, so the gate short-circuited and apply never
// rewrote it — leaving the tool with a symlink dangling at a file nothing would
// ever recreate, while plan/status/doctor all reported healthy.
func TestApplyRestoresDeletedRenderedVariant(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	writeConfig(t, repo, "first/model-a")

	e := buildEngine(t, home, repo)
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	variant := filepath.Join(e.SubagentDir(), "onto-reviewer.opencode.md")
	if err := os.Remove(variant); err != nil {
		t.Fatal(err)
	}

	e2 := buildEngine(t, home, repo)
	if err := e2.Apply(mustPlan(t, e2)); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if _, err := os.Stat(variant); err != nil {
		t.Fatalf("rendered variant not restored by apply: %v", err)
	}
}

// ontoFrameworkTOML installs the onto framework, whose `onto` agent is
// OpenCode-primary — agentfm renders no Claude variant for it by design.
const ontoFrameworkTOML = `
[frameworks.onto]
source = "builtin:onto"
scope = "project"

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"

[models.opencode.architectural]
model = "some/architectural-model"
variant = "high"
[models.opencode.coding]
model = "some/coding-model"
effort = "medium"
[models.opencode.trivial]
model = "some/trivial-model"
effort = "medium"
`

// TestDoctorSilentOnPrimaryAgentClaudeVariant guards against a permanent false
// positive: `onto` is an OpenCode-primary agent, so agentfm deliberately renders
// no Claude variant and the adapter deliberately does not project it. doctor
// fell back to the shared anchor, found it, then demanded a Claude link that
// must never exist — reporting `subagent "onto" content present, not linked for
// claude (run apply)` on every healthy workspace. Worse, the advice was a dead
// end: apply correctly does nothing, so the warning could never be cleared.
func TestDoctorSilentOnPrimaryAgentClaudeVariant(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(ontoFrameworkTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	e := buildEngine(t, home, repo)
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("apply: %v", err)
	}

	e2 := buildEngine(t, home, repo)
	for _, line := range e2.Doctor() {
		if strings.Contains(line, `subagent "onto"`) && strings.Contains(line, "claude") &&
			strings.HasPrefix(line, "warn:") {
			t.Fatalf("doctor reports an unfixable finding for the primary agent's absent Claude variant: %q", line)
		}
	}

	// The OpenCode side must still be reported healthy — silencing the Claude
	// false positive must not blind doctor to the variant that IS projected.
	var sawOpenCode bool
	for _, line := range e2.Doctor() {
		if strings.Contains(line, `subagent "onto"`) && strings.Contains(line, "opencode") {
			sawOpenCode = true
			if !strings.HasPrefix(line, "ok:") {
				t.Fatalf("opencode side of the primary agent not healthy: %q", line)
			}
		}
	}
	if !sawOpenCode {
		t.Fatal("doctor said nothing about the primary agent's OpenCode projection")
	}
}

// TestSubagentRenderFingerprintDistinguishesRoutes pins the fingerprint's job:
// it must change when a route changes and stay put when nothing does. A
// fingerprint that collided across route sets would silently skip the re-render
// this whole gate exists to trigger.
func TestSubagentRenderFingerprintDistinguishesRoutes(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	writeConfig(t, repo, "first/model-a")
	a := buildEngine(t, home, repo).subagentRenderContext()
	// Built independently from the same config: the fingerprint must not depend
	// on map iteration order, or every apply would needlessly re-materialize.
	aAgain := buildEngine(t, home, repo).subagentRenderContext()
	writeConfig(t, repo, "second/model-b")
	b := buildEngine(t, home, repo).subagentRenderContext()

	if subagentRenderFingerprint(a) == subagentRenderFingerprint(b) {
		t.Fatal("fingerprint collided across different model routes: a route change would not re-render")
	}
	if subagentRenderFingerprint(a) != subagentRenderFingerprint(aAgain) {
		t.Fatal("fingerprint is not stable for identical routes: every apply would re-materialize")
	}
}
