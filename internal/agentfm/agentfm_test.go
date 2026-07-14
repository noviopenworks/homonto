package agentfm

import (
	"strings"
	"testing"
)

const readOnlyReviewer = `---
name: code-reviewer
description: Use to review a diff; reports findings ranked by severity.
mode: subagent
homonto:
  read_only: true
  dialogs: true
---
You are a focused code reviewer.
`

const readOnlyExplorer = `---
name: codebase-explorer
description: Answer how a codebase works.
mode: subagent
homonto:
  read_only: true
  bash: false
  dialogs: true
---
Explore.
`

func TestNeedsTransform(t *testing.T) {
	if !NeedsTransform([]byte(readOnlyReviewer)) {
		t.Fatal("reviewer with homonto block should need transform")
	}
	if NeedsTransform([]byte("---\nname: x\ndescription: y\n---\nbody\n")) {
		t.Fatal("no homonto block should not need transform")
	}
	if NeedsTransform([]byte("no frontmatter at all")) {
		t.Fatal("no frontmatter should not need transform")
	}
}

func TestRenderClaude_ReadOnlyReviewer(t *testing.T) {
	out, err := Render([]byte(readOnlyReviewer), "claude")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	// Claude gets a tools allowlist, no permission block, no homonto block.
	if !strings.Contains(s, "tools: Read, Grep, Glob, Bash") {
		t.Errorf("claude tools allowlist missing:\n%s", s)
	}
	if strings.Contains(s, "permission:") {
		t.Errorf("claude output must not carry an OpenCode permission block:\n%s", s)
	}
	if strings.Contains(s, "homonto:") {
		t.Errorf("neutral homonto block must be stripped:\n%s", s)
	}
	// Original lines and body preserved verbatim.
	if !strings.Contains(s, "name: code-reviewer") ||
		!strings.Contains(s, "description: Use to review a diff; reports findings ranked by severity.") ||
		!strings.Contains(s, "You are a focused code reviewer.") {
		t.Errorf("name/description/body not preserved:\n%s", s)
	}
}

func TestRenderOpenCode_ReadOnlyReviewer(t *testing.T) {
	out, err := Render([]byte(readOnlyReviewer), "opencode")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "permission:") ||
		!strings.Contains(s, "  edit: deny") ||
		!strings.Contains(s, "  question: allow") {
		t.Errorf("opencode permission block missing edit/question:\n%s", s)
	}
	if strings.Contains(s, "tools:") {
		t.Errorf("opencode output must not carry a Claude tools string:\n%s", s)
	}
	if strings.Contains(s, "homonto:") {
		t.Errorf("neutral homonto block must be stripped:\n%s", s)
	}
	// Reviewer keeps bash (not denied).
	if strings.Contains(s, "bash: deny") {
		t.Errorf("reviewer should not deny bash:\n%s", s)
	}
}

func TestRenderExplorer_BashDenied(t *testing.T) {
	claude, _ := Render([]byte(readOnlyExplorer), "claude")
	if strings.Contains(string(claude), "Bash") {
		t.Errorf("explorer denies bash, so Claude allowlist must omit Bash:\n%s", claude)
	}
	if !strings.Contains(string(claude), "tools: Read, Grep, Glob") {
		t.Errorf("explorer claude allowlist wrong:\n%s", claude)
	}
	oc, _ := Render([]byte(readOnlyExplorer), "opencode")
	if !strings.Contains(string(oc), "  bash: deny") {
		t.Errorf("explorer opencode must deny bash:\n%s", oc)
	}
}

func TestRender_NoHomontoBlock_Unchanged(t *testing.T) {
	in := "---\nname: x\ndescription: y\nmode: subagent\n---\nbody\n"
	out, err := Render([]byte(in), "claude")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Errorf("content without a homonto block must be returned unchanged\n got: %q\nwant: %q", out, in)
	}
}

func TestRender_UnknownTool(t *testing.T) {
	if _, err := Render([]byte(readOnlyReviewer), "codex"); err == nil {
		t.Fatal("unknown tool should error")
	}
}
