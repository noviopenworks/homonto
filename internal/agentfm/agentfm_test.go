package agentfm

import (
	"strings"
	"testing"
)

// A read-only specialist: no edits, no shell, dialogs on, spawns nothing, and a
// role that stamps a model from the render context.
const readOnlyReviewer = `---
name: onto-reviewer
description: Use to review a diff; reports findings ranked by severity.
mode: subagent
homonto:
  role: architectural
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
  role: architectural
  primary: true
  steps: 60
  spawn: [onto-implementer, onto-reviewer]
---
Drive the workflow.
`

func ctx() RenderContext {
	return RenderContext{Roles: map[string]ModelSpec{
		"architectural": {Model: "opus"}, "coding": {Model: "sonnet"}, "trivial": {Model: "haiku"},
	}}
}

func mustRender(t *testing.T, content, tool string) string {
	t.Helper()
	out, err := Render("agent", []byte(content), tool, ctx())
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
	// read-only + spawn:[] → allowlist without Edit/Write and without Task.
	if !strings.Contains(s, "tools: Read, Grep, Glob, Bash\n") {
		t.Errorf("claude tools wrong:\n%s", s)
	}
	if strings.Contains(s, "Task") || strings.Contains(s, "Edit") || strings.Contains(s, "Write") {
		t.Errorf("read-only non-spawning agent must not carry Task/Edit/Write:\n%s", s)
	}
	if !strings.Contains(s, "model: opus\n") {
		t.Errorf("role architectural must stamp model: opus:\n%s", s)
	}
	if strings.Contains(s, "permission:") || strings.Contains(s, "homonto:") {
		t.Errorf("claude output must not carry permission/homonto:\n%s", s)
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
	oc := mustRender(t, orchestrator, "opencode")
	if !strings.Contains(oc, "mode: primary") || !strings.Contains(oc, "steps: 60") {
		t.Errorf("opencode primary must carry mode: primary + steps:\n%s", oc)
	}
	// named spawn → task glob allowlist.
	for _, want := range []string{"  task:", `    "*": deny`, `    "onto-implementer": allow`, `    "onto-reviewer": allow`} {
		if !strings.Contains(oc, want) {
			t.Errorf("opencode spawn topology missing %q:\n%s", want, oc)
		}
	}
	// Claude view of a NON-primary agent with the same named spawn: Task present
	// (advisory), since Claude cannot scope to specific agents.
	named := strings.Replace(orchestrator, "  primary: true\n  steps: 60\n", "", 1)
	if cl := mustRender(t, named, "claude"); !strings.Contains(cl, "Task") {
		t.Errorf("named-spawn agent should keep Task in Claude (advisory):\n%s", cl)
	}
}

func TestRender_NoHomontoBlock_Unchanged(t *testing.T) {
	in := "---\nname: x\ndescription: y\nmode: subagent\n---\nbody\n"
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
// it cannot always do: the merged model (tier + override) is only known here.
func TestRenderClaude_VariantOnFullModelIDErrors(t *testing.T) {
	ctx := RenderContext{Roles: map[string]ModelSpec{
		"architectural": {Model: "claude-opus-4-8", Variant: "1m"},
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

func TestRender_MissingRouteOmitsModel(t *testing.T) {
	// role set but the render context has no model for it → no model line.
	out := func() string {
		b, _ := Render("onto-reviewer", []byte(readOnlyReviewer), "claude", RenderContext{})
		return string(b)
	}()
	if strings.Contains(out, "model:") {
		t.Errorf("missing route must omit model:\n%s", out)
	}
}

// TestRenderUnknownRoleErrors: an unknown role would look up no tier and render
// the agent with no model line — silently weaker than declared. Render must
// fail naming the agent and the bad role instead.
func TestRenderUnknownRoleErrors(t *testing.T) {
	content := "---\nname: rev\ndescription: d\nmode: subagent\nhomonto:\n  role: reviewing\n---\nbody\n"
	for _, tool := range []string{"claude", "opencode"} {
		_, err := Render("rev", []byte(content), tool, ctx())
		if err == nil {
			t.Fatalf("Render(%s): unknown role must error", tool)
		}
		if !strings.Contains(err.Error(), `"reviewing"`) || !strings.Contains(err.Error(), `"rev"`) {
			t.Fatalf("Render(%s): error must name the agent and role, got: %v", tool, err)
		}
	}
}
