# Project Development Instructions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one project-local, shared development workflow for OpenCode and Claude Code.

**Architecture:** `AGENTS.md` is the canonical repository instruction document. `CLAUDE.md` contains only an import of that document, so both assistants receive the same workflow without duplicated policy.

**Tech Stack:** Markdown, CodeGraph CLI, Go test tooling.

## Global Constraints

- Keep the workflow project-local in root `AGENTS.md` and `CLAUDE.md`.
- Preserve `AGENTS.md` as the only source of development-policy text.
- Require CodeGraph before manual code exploration when `.codegraph/` exists; do not block if it does not.
- Keep Graphify opt-in for cross-cutting analysis.
- Do not claim verification passed without command output.

---

### Task 1: Add Canonical Development Instructions

**Files:**
- Create: `AGENTS.md`
- Create: `CLAUDE.md`
- Test: root instruction-file validation

**Interfaces:**
- Consumes: the contributor workflow in `README.md` and the approved design in `docs/superpowers/specs/2026-07-10-project-development-instructions-design.md`.
- Produces: root `AGENTS.md` as the shared policy and `CLAUDE.md` importing it for Claude Code.

- [x] **Step 1: Create the shared workflow document**

Write `AGENTS.md` with these requirements:

```markdown
# Development Instructions

Start new development with `/comet`; inspect active change state before
starting separate change work. Treat `openspec/changes/` as active work,
`docs/superpowers/` as design and implementation planning, and
`docs/changes/` as historical only.

When `.codegraph/` exists, use CodeGraph before grep, glob, or direct reads to
locate and understand code. If it is absent or unavailable, continue with the
repository's normal inspection tools. Use Graphify only for broad architecture,
documentation, or cross-cutting analysis.

Read the relevant capability specs, ADRs, and nearby implementation before
changing behavior. Keep changes focused. Do not revert unrelated user work.

For behavior changes, add or update focused tests and run the narrowest useful
verification command. Before reporting completion, state the command result
and any verification gap.
```

- [x] **Step 2: Create the Claude Code entry point**

Write `CLAUDE.md` exactly as:

```markdown
@AGENTS.md
```

- [x] **Step 3: Verify the instruction structure**

Run: `test -f AGENTS.md && test "$(< CLAUDE.md)" = '@AGENTS.md' && rg -F 'Start new development with `/comet`' AGENTS.md`

Expected: exit status `0`; the command prints the `/comet` instruction.

- [x] **Step 4: Verify project tests remain healthy**

Run: `go test ./...`

Expected: exit status `0`.

- [x] **Step 5: Commit the instruction files and planning artifacts**

Run:

```bash
git add AGENTS.md CLAUDE.md \
  docs/superpowers/specs/2026-07-10-project-development-instructions-design.md \
  docs/superpowers/plans/2026-07-10-project-development-instructions.md
git commit -m "docs: add project development instructions"
```

Expected: one commit containing only the instruction files and their approved design and implementation plan.

## Self-Review

- Spec coverage: Task 1 implements the canonical `AGENTS.md`, importing
  `CLAUDE.md`, CodeGraph fallback, opt-in Graphify use, focused verification,
  and explicit verification reporting.
- Placeholder scan: no TBDs, incomplete requirements, or undefined interfaces.
- Type consistency: not applicable; this change has no programmatic types.
