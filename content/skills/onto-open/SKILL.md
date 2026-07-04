---
name: onto-open
description: onto phase 1 — open a change. Use when starting a new change or when the dispatcher routes here (phase open) — clarifies requirements, checks for scope splits, and creates the change workspace with proposal and tasks skeleton.
---

# onto-open — Phase 1: Open

Turn an idea, feature request, or problem statement into a change workspace
with an unambiguous proposal. Nothing is designed and nothing is built here.

## Entry check

- No workspace exists yet for this work, **or** the workspace's `state.yaml`
  says `phase: open` with `workflow: full`.
- Bug fixes and small tweaks belong to `onto-fix` / `onto-tweak` — if the
  request fits a preset, hand over to it instead.
- Any other state → route back through `/onto` (the dispatcher rederives the
  real phase).

## Steps

### 1. Clarify

Ask questions until the requirement is unambiguous — one topic at a time,
multiple-choice where possible. Do not treat a single Q&A round as enough
for anything non-trivial. Ground every claim about the existing codebase in
graphify/codegraph queries (never guesswork; the dispatcher's preflight
guarantees the tooling exists).

The clarification must end in a summary covering:

- **Goals** — the problem actually being solved, expected outcome
- **Non-goals** — explicitly out of scope
- **Scope boundaries** — modules/users/platforms/data in and out
- **Key unknowns** — open assumptions, risks, dependencies
- **Draft acceptance scenarios** — core success path + important edge cases

> **GATE (clarification complete):** present the summary and ask the user to
> confirm it before creating any artifact. Always requires fresh input —
> a blanket "run to completion" directive does NOT pre-answer this gate.

### 2. Split preflight

If the request spans multiple independent capabilities, journeys, or
milestones — anything that could be designed, built, verified, and closed
independently — propose a split: per item, a name, goals, non-goals,
dependencies, and core scenarios.

> **GATE (split decision, only when a split is proposed):** the user chooses
> "split into separate changes", "keep as one (record why)", or "adjust the
> split". Each accepted item becomes its own change via this skill. Always
> fresh input.

### 3. Create the workspace

Create `docs/changes/<name>/` (name confirmed by the user, kebab-case):

- `state.yaml` — schema per `docs/changes/README.md`: `change: <name>`,
  `workflow: full`, `phase: open`, `created: <today>`, `base_ref: <current
  git sha>`, `decisions:` all null, `verify: {mode: null, result: pending}`,
  `guides: pending`, `archived: false`.
- `proposal.md` — **why** (problem, motivation), **what changes** (bulleted,
  breaking changes marked), **capability impact** (which `docs/specs/`
  capabilities are new or modified — check the existing spec files), and
  **impact** (code, dependencies, systems).
- `tasks.md` — unchecked checklist skeleton grouped by area (foundation /
  implementation / integration / validation). Tasks get refined in build;
  here they set boundaries.

Everything in the proposal must trace back to the confirmed clarification
summary — no invented scope.

> **GATE (artifact review):** summarize proposal + tasks skeleton and ask
> the user to approve or request adjustments. Iterate until approved.
> Always fresh input.

## Exit checklist

- [ ] Workspace exists with `state.yaml`, `proposal.md`, `tasks.md`, all
      non-empty and consistent with the confirmed summary
- [ ] Both gates answered by the user
- [ ] `state.yaml` phase advanced: `open → design`
- [ ] Announce the transition and load `onto-design`
