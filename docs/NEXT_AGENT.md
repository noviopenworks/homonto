# Handoff: catalog-foundation-skills

**Date:** 2026-07-09
**Comet change:** `catalog-foundation-skills` (phase: design, workflow: full)
**Active branch:** `main`
**Base ref:** `5eeff35`

## What is done

1. **Comet migration complete and committed on main** (5 commits):
   - `a569675 chore: bootstrap openspec and comet config`
   - `f687314 chore: dogfood comet development skills`
   - `1555d0b docs: route development workflow through comet`
   - `feec0c9 docs: specify comet as development workflow`
   - `5eeff35 docs: record comet migration verification`

2. **OpenSpec change `catalog-foundation-skills` created** with all open-phase artifacts:
   - `openspec/changes/catalog-foundation-skills/proposal.md` -- WHY + WHAT
   - `openspec/changes/catalog-foundation-skills/design.md` -- high-level architecture
   - `openspec/changes/catalog-foundation-skills/tasks.md` -- 7 task groups, 28 tasks
   - `openspec/changes/catalog-foundation-skills/specs/{builtin-catalog,framework-expansion,config-model,tool-adapters}/spec.md` -- delta specs with SHALL requirements and GIVEN/WHEN/THEN scenarios
   - `.comet.yaml` state file at `phase: design`

3. **Comet design handoff generated**:
   - `openspec/changes/catalog-foundation-skills/.comet/handoff/design-context.{json,md}`
   - `handoff_hash` recorded in `.comet.yaml`

4. **Exploration complete** -- tool format investigation, PRD split decided:
   - Change A (this one): catalog + framework expansion + builtin skill projection
   - Change B (later): command projection
   - Change C (later): subagent projection

## Where it stopped

**Phase: design, step: brainstorming (comet-design Step 1b)**

The brainstorming skill was loaded. One key technical decision was presented to the user but the question was dismissed (user asked to write this handoff instead):

**Open decision: go:embed layout for the bundled catalog**

Go's `//go:embed` resolves paths relative to the package directory and does not allow `..`. Two options were proposed:

- **Option C (recommended):** `catalog/` at repo root as its own Go package (`package catalog`) with `//go:embed all:frameworks all:skills`. `internal/catalog/` imports the embedded FS from it. Keeps source tree at repo root matching the design doc.
- **Option A:** Catalog content under `internal/catalog/content/`. Idiomatic Go but buries content in internal/.

This decision must be resolved before the Design Doc can be written.

## Key confirmed decisions (from open phase)

- **Catalog storage:** go:embed + materialize to `.homonto/catalog/skills/<name>/`
- **Catalog content:** all 4 first-release frameworks (onto, comet, superpowers, openspec)
- **Scope:** skills only (commands and subagents are separate changes B and C)
- **Framework metadata:** TOML (`catalog/frameworks/<name>/framework.toml`) with name, version, dependencies, skills table
- **Expansion:** `[frameworks.comet] source = "builtin:comet"` expands to all comet skills + transitive deps (superpowers, openspec)
- **Materialization:** version-aware, gated on `.homonto/state.json` catalog version
- **State:** `.homonto/catalog/` is generated cache (gitignored)
- **Split:** 3 changes agreed -- A (this), B (commands), C (subagents)

## Tool format findings (for changes B and C later)

```
Skills     Claude: ~/.claude/skills/<n>/    OpenCode: ~/.config/opencode/skills/<n>/
Commands   Claude: ~/.claude/commands/<n>.md  OpenCode: ~/.config/opencode/command/<n>.md
Subagents  Claude: ~/.claude/agents/ (unclear)  OpenCode: opencode.jsonc "agent" key (JSON merge)
```

## How to resume

1. **Run `/comet`** -- it will detect the active `catalog-foundation-skills` change at `phase: design` and route to `/comet-design`.
2. **Resolve the go:embed layout decision** (Option C recommended).
3. **Continue brainstorming** to finalize remaining technical details, then confirm design proposal (blocking point).
4. **Write Design Doc** to `docs/superpowers/specs/2026-07-09-catalog-foundation-skills-design.md`.
5. **Run design guard** to advance to build phase.
6. **In build phase:** choose isolation/execution/TDD/review mode, write implementation plan, execute tasks.

## Comet script bootstrap (run once on resume)

```bash
COMET_ENV="$(find . "$HOME"/.*/skills "$HOME/.config" "$HOME/.gemini" -path '*/comet/scripts/comet-env.mjs' -type f -print -quit 2>/dev/null)"
COMET_SCRIPTS_DIR="$(node "$COMET_ENV")"
COMET_STATE="$COMET_SCRIPTS_DIR/comet-state.mjs"
COMET_GUARD="$COMET_SCRIPTS_DIR/comet-guard.mjs"
COMET_HANDOFF="$COMET_SCRIPTS_DIR/comet-handoff.mjs"
```

## Current verification state

- `go test ./... -count=1` -- 168 passed (pre-change baseline)
- `go vet ./...` -- clean
- `go build ./...` -- success
- `go run . status` -- `No drift.`
- `openspec list --json --no-color` -- `[catalog-foundation-skills]`
- Worktree: clean on `main`
