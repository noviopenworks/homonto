package agentfm

import (
	"strings"
	"testing"
)

// A read-only specialist: no edits, no shell, dialogs on, spawns nothing. The
// model is supplied by the render context's Overrides, mirroring a
// [subagents.<name>.<tool>] block in homonto.toml.
const readOnlyReviewer = `---
name: onto-reviewer
description: Use to review a diff; reports findings ranked by severity.
mode: subagent
homonto:
  read_only: true
  dialogs: true
  spawn: []
---
You are a focused code reviewer.
`

// An orchestrator: may edit, may spawn a fixed set, is the OpenCode primary.
const orchestrator = `---
name: onto
description: dispatcher
mode: subagent
homonto:
  primary: true
  steps: 60
  spawn: [onto-implementer, onto-reviewer]
---
Drive the workflow.
`

func ctx() RenderContext {
	return RenderContext{Overrides: map[string]ModelSpec{
		"onto-reviewer": {Model: "opus"},
		"onto":          {Model: "opus"},
	}}
}

func mustRender(t *testing.T, content, tool string) string {
	t.Helper()
	out, err := Render("onto-reviewer", []byte(content), tool, ctx())
	if err != nil {
		t.Fatalf("Render(%s): %v", tool, err)
	}
	return string(out)
}

func TestNeedsTransform(t *testing.T) {
	if !NeedsTransform([]byte(readOnlyReviewer)) {
		t.Fatal("homonto block should need transform")
	}
	if NeedsTransform([]byte("---\nname: x\ndescription: y\n---\nbody\n")) {
		t.Fatal("no homonto block should not need transform")
	}
}

func TestRenderClaude_ReadOnlyReviewer(t *testing.T) {
	s := mustRender(t, readOnlyReviewer, "claude")
	// read-only + spawn:[] → a denylist covering exactly the denied intent;
	// everything else keeps Claude's defaults (no allowlist stripping them).
	if !strings.Contains(s, "disallowedTools: Edit, Write, NotebookEdit, Agent, Task\n") {
		t.Errorf("claude denylist wrong:\n%s", s)
	}
	if strings.Contains(s, "tools:") {
		t.Errorf("claude output must deny by exception, not allowlist:\n%s", s)
	}
	// Claude has no mode: field — emitting one is unrecognized noise.
	if strings.Contains(s, "mode:") {
		t.Errorf("claude output must not carry mode:\n%s", s)
	}
	if !strings.Contains(s, "model: opus\n") {
		t.Errorf("override model must stamp model: opus:\n%s", s)
	}
	if strings.Contains(s, "permission:") || strings.Contains(s, "homonto:") {
		t.Errorf("claude output must not carry permission/homonto:\n%s", s)
	}
}

func TestRenderClaude_EditCapableWorker(t *testing.T) {
	// read_only: false + bash allowed + spawn:[] → only spawning is denied.
	worker := strings.Replace(readOnlyReviewer, "  read_only: true\n", "", 1)
	s := mustRender(t, worker, "claude")
	if !strings.Contains(s, "disallowedTools: Agent, Task\n") {
		t.Errorf("worker must deny only spawning:\n%s", s)
	}
}

func TestRenderClaude_StepsBecomeMaxTurns(t *testing.T) {
	// steps on a NON-primary agent renders as Claude's maxTurns.
	bounded := strings.Replace(readOnlyReviewer, "homonto:\n", "homonto:\n  steps: 40\n", 1)
	s := mustRender(t, bounded, "claude")
	if !strings.Contains(s, "maxTurns: 40\n") {
		t.Errorf("steps must render as Claude maxTurns:\n%s", s)
	}
}

func TestRenderOpenCode_ReadOnlyReviewer(t *testing.T) {
	s := mustRender(t, readOnlyReviewer, "opencode")
	for _, want := range []string{"mode: subagent", "model: opus", "permission:", "  edit: deny", "  question: allow", "  task: deny"} {
		if !strings.Contains(s, want) {
			t.Errorf("opencode output missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "tools:") || strings.Contains(s, "homonto:") {
		t.Errorf("opencode output must not carry a Claude tools string / homonto:\n%s", s)
	}
}

func TestRenderPrimary_ClaudeSkipped_OpenCodeMode(t *testing.T) {
	out, err := Render("onto", []byte(orchestrator), "claude", ctx())
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Errorf("a primary agent must have no Claude variant, got:\n%s", out)
	}
	oc, err := Render("onto", []byte(orchestrator), "opencode", ctx())
	if err != nil {
		t.Fatalf("Render(opencode) primary: %v", err)
	}
	s := string(oc)
	if !strings.Contains(s, "mode: primary") || !strings.Contains(s, "steps: 60") {
		t.Errorf("opencode primary must carry mode: primary + steps:\n%s", s)
	}
	// named spawn → task glob allowlist.
	for _, want := range []string{"  task:", `    "*": deny`, `    "onto-implementer": allow`, `    "onto-reviewer": allow`} {
		if !strings.Contains(s, want) {
			t.Errorf("opencode spawn topology missing %q:\n%s", want, s)
		}
	}
	// Claude view of a NON-primary agent with the same named spawn: spawning
	// stays available (Agent/Task not denied) — the named list is advisory in
	// Claude, enforced in OpenCode.
	named := strings.Replace(orchestrator, "  primary: true\n  steps: 60\n", "", 1)
	if cl := mustRender(t, named, "claude"); strings.Contains(cl, "Agent, Task") {
		t.Errorf("named-spawn agent must not deny spawning in Claude:\n%s", cl)
	}
}

// dialogs is enforced BOTH ways in OpenCode: false must render question: deny,
// or the "subagents never prompt" protocol is silently unenforced there.
func TestRenderOpenCode_NoDialogsDeniesQuestion(t *testing.T) {
	silent := strings.Replace(readOnlyReviewer, "  dialogs: true\n", "", 1)
	s := mustRender(t, silent, "opencode")
	if !strings.Contains(s, "  question: deny") {
		t.Errorf("dialogs:false must render question: deny:\n%s", s)
	}
}

func TestRender_NoHomontoBlock_Unchanged(t *testing.T) {
	in := "---\nname: x\ndescription: y\nmode: subagent\n---\nbody\n"
	// A missing homonto block returns before model context is considered.
	if out := mustRender(t, in, "claude"); out != in {
		t.Errorf("content without a homonto block must be unchanged\n got: %q", out)
	}
}

func TestRender_UnknownTool(t *testing.T) {
	if _, err := Render("onto-reviewer", []byte(readOnlyReviewer), "codex", ctx()); err == nil {
		t.Fatal("unknown tool should error")
	}
}

// A variant on a non-alias model has no Claude spelling. The render used to
// silently drop the variant — shipping an agent quietly weaker than declared —
// on the assumption that config validation had rejected the combination, which
// it cannot always do: the override is judged against its own model, but a
// future caller that bypasses Load could supply an unrenderable combination.
func TestRenderClaude_VariantOnFullModelIDErrors(t *testing.T) {
	ctx := RenderContext{Overrides: map[string]ModelSpec{
		"onto-reviewer": {Model: "claude-opus-4-8", Variant: "1m"},
	}}
	_, err := Render("onto-reviewer", []byte(readOnlyReviewer), "claude", ctx)
	if err == nil || !strings.Contains(err.Error(), "alias") {
		t.Fatalf("a variant on a full model id must error loudly, got: %v", err)
	}
	// The same spec renders fine for OpenCode, where variant is a real field.
	oc, err := Render("onto-reviewer", []byte(readOnlyReviewer), "opencode", ctx)
	if err != nil || !strings.Contains(string(oc), "variant: 1m") {
		t.Fatalf("opencode must carry the variant as its own field, got err=%v:\n%s", err, oc)
	}
}

// TestRenderNoModelErrors: the renderer backstops config validation for the
// narrow case where an override entry IS supplied but its Model is blank (a
// gap load-time validation can miss for framework-expanded agents). An empty
// render context (no entry at all) is treated leniently: the variant renders
// without a model line, so the catalog's verbatim-materialize unit tests work
// and the engine can render an untargeted-tool variant the adapter will skip
// by target filter anyway.
func TestRenderNoModelErrors(t *testing.T) {
	for _, tool := range []string{"claude", "opencode"} {
		ctx := RenderContext{Overrides: map[string]ModelSpec{
			"ghost": {Variant: "1m"}, // entry present but no Model
		}}
		_, err := Render("ghost", []byte(readOnlyReviewer), tool, ctx)
		if err == nil {
			t.Fatalf("Render(%s): an override entry with no model must error", tool)
		}
		if !strings.Contains(err.Error(), `"ghost"`) || !strings.Contains(err.Error(), tool) {
			t.Fatalf("Render(%s): error must name the agent and tool, got: %v", tool, err)
		}
		if !strings.Contains(err.Error(), "[subagents.ghost."+tool+"]") {
			t.Fatalf("Render(%s): error must name the block to add, got: %v", tool, err)
		}
	}
}
