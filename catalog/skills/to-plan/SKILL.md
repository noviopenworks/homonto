---
name: to-plan
description: to phase 1 — plan. Use when an active change has phase plan — writes docs/tasks/<name>/plan.md as a short list of bite-sized, verifiable tasks, then advances the change to do.
---

# to-plan — Phase 1: Plan

Turn the request into a short, executable plan. The plan is a reviewable git
artifact — write it for the person who reads the PR, not for yourself.

## Entry check

- `to status --json` shows the change at `phase: plan`.
- If `plan.md` already has content, a previous session started planning —
  read it and continue rather than starting over.

## Steps

1. **Understand before writing.** Ground every claim about the codebase in
   reading. For questions that span many files, dispatch `to-explorer` (one at
   a time, never in parallel) and work from its conclusions.
2. **Suggest isolation.** Recommend a branch for the change (the binary is
   git-blind and will not check; this is process advice, not a gate). The user
   may decline — proceed either way.
3. **Write `docs/tasks/<name>/plan.md`:**
   - A two-or-three-sentence statement of the goal and the approach.
   - A task list: each task bite-sized (one sitting, one concern), naming the
     files it touches and the specific command that verifies it. Use `- [ ]`
     checkboxes so `do` can track completion.
   - A "verify" line at the bottom: the narrowest command that proves the whole
     change works.
4. **De-slop it.** Run the `to-no-slop` rules over the plan prose.
5. **Confirm scope with the user** if the plan grew beyond what they asked —
   otherwise proceed.
6. **Advance:** `to phase <name>`. The change is now at `do`; hand off to the
   `to-do` skill.

## Rules

- Keep the plan under a screen where possible. A plan nobody reads is
  ceremony, and ceremony is what to exists to avoid.
- Never hand-edit `to-state.yaml`; the binary owns it.
- If the work turns out to need evidence-gated phases, spec deltas, or a
  dependency graph, say so: that is onto-shaped work, and this repo chose to.
  Do not rebuild onto inside a plan.md.
