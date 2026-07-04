---
name: onto-design
description: onto phase 2 — deep design. Use when an active full-workflow change has phase design — brainstorming-grade exploration, approach confirmation, then design.md plus ADR drafts and spec deltas.
---

# onto-design — Phase 2: Deep Design

Produce a confirmed technical design before any implementation exists.
**Design cannot be skipped in the full workflow** — this phase is the reason
the full workflow exists.

## Entry check

- `state.yaml` has `phase: design` and `workflow: full`; `proposal.md`
  exists and was approved in open.
- Presets never enter this phase — except a preset **upgraded** to full,
  which arrives here to backfill the design it skipped.
- Any other state → route back through `/onto`.

## Steps

### 1. Explore ground truth

Before proposing anything, read the real system: graphify/codegraph queries
for structure and call paths, then the actual files. Map the integration
points the proposal touches. Never design against an imagined codebase.

### 2. Question until clear

If goals, scope, constraints, or acceptance scenarios still have gaps, keep
asking — one question at a time. Do not write a design around an unresolved
unknown; resolve it or explicitly record it as a risk with a fallback.

### 3. Propose 2–3 approaches

Present genuinely different candidate approaches with trade-offs and a
recommendation. Lead with the recommended one and say why.

> **GATE (approach confirmation):** the user picks or adjusts an approach.
> The final `design.md` MUST NOT be written before this gate is answered.
> Always fresh input — a blanket directive does not pre-answer it.

**No implementation code in this phase.** Writing source code before a
confirmed design exists is prohibited, regardless of how simple the change
looks.

### 4. Write the design artifacts

After confirmation, write into the workspace:

- `design.md` — summary, goals/non-goals, architecture (diagrams welcome),
  key decisions with the alternatives rejected, data flow, error handling,
  testing strategy. First lines after the title:
  `Status: Confirmed` and `Confirmed: <date>` (the dispatcher's phase
  derivation treats only a confirmed design.md as design-complete).
- `adr/<slug>.md` — one draft per significant decision (template:
  `docs/adr/README.md`), `Status: Proposed`, **unnumbered**. Numbers are
  assigned at close.
- `specs/<capability>.md` — delta specs (format: `docs/specs/README.md`):
  `## ADDED|MODIFIED|REMOVED Requirements` with SHALL statements and
  GIVEN/WHEN/THEN scenarios. Every behavior change in the design needs a
  scenario here — these are what verify will demand evidence for. Deltas are
  living documents: build may refine them; close merges them.

Update `tasks.md` if the design revealed different task boundaries.

## Exit checklist

- [ ] `design.md` exists, marked `Status: Confirmed` with date, and matches
      the user-confirmed approach
- [ ] An ADR draft exists for every significant decision named in design.md
- [ ] A delta spec scenario exists for every behavior change
- [ ] No implementation code was written
- [ ] `state.yaml` phase advanced: `design → build`
- [ ] Announce the transition and load `onto-build`
