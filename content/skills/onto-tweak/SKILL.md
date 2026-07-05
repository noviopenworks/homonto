---
name: onto-tweak
description: onto preset — small non-bug change. Use for copy, configuration, documentation, or prompt tweaks, and for small features within tweak limits (≤5 files, no new capability, no existing-spec requirement change) — open-lite, lightweight build, light verify, close; upgrades to the full workflow when scope grows.
---

# onto-tweak — Preset: Small Change

Fast path for small non-bug changes (copy, config values, docs, prompts)
and for small features that stay within the tweak limits:
**open-lite → lightweight build → light verify → close**. Skips design and
the full plan — bounded by strict upgrade rules.

## Entry check

- A small, local, non-bug change request, or an active change with
  `workflow: tweak`. This preset owns the change's whole lifecycle.
- Broken behavior → `onto-fix`. **Small features are tweak territory** when
  ALL of: ≤5 files touched (test files excluded), no new capability (no new
  `docs/specs/` file), and no existing spec's requirements change.
  Structural work or anything introducing a new capability → full workflow
  via `onto-open`.
- Read `notes.md` at entry when present (recommended for any tweak that
  spans sittings). If any skill's `references/` directory is missing,
  degrade per the dispatcher rule: reconstruct from the `docs/` contract
  pointers, note the gap, continue.

## Steps

### 1. Open-lite

One-paragraph `proposal.md` — first line `Preset: tweak` (the dispatcher's
state rebuild keys on this marker), then what + why — plus short
`tasks.md`, and `state.yaml` with `workflow: tweak`, `phase: build`,
`created`, `base_ref`, `guides: pending`, and `decisions` defaulted at
open-lite (`isolation: branch`, `execution: direct`, `tdd: direct`)
(canonical schema: `onto/references/state-yaml.md`; artifact templates:
`onto-open/references/`). Branch: `tweak/YYYYMMDD/<name>`. Stamp
`metrics.phases.<phase>` at each phase exit.

### 2. Lightweight build

No `plan.md` required. Still binding:

- one commit per task, checked off in `tasks.md` as it lands
- on ANY failure: systematic debugging — root cause before any fix
- stay inside the tweak's stated scope; anything more hits the upgrade gate

### 3. Light verify

Demonstrate the changed behavior/content with a fresh command + output
(render the doc, run the config consumer, show the diff taking effect) and
run the regression suite. Write `docs/changes/<name>/verification.md`
(template: `onto-verify/references/verification.md`) — brief is fine,
absent is not. One adversarial skeptic optional (skips recorded).
`verify.result` set; failures hit the same fix-or-accept gate as the full
workflow.

### 4. Close

Full `onto-close` obligations: lint, merge any spec deltas, guides
`updated` or `"waived: <reason>"`, metrics finalized, final confirmation,
archive, ship handoff offered.

## Upgrade rules

> **GATE (upgrade):** pause, explain the trigger, and require fresh user
> confirmation to upgrade to the full workflow when ANY of:
>
> - the change touches **more than 5 files** (test files excluded — the
>   entry limit is ≤5, so exactly 5 is still a tweak)
> - cross-module coordination is required
> - **5+ new test cases** are needed
> - config **keys are added or removed** (value changes are fine)
> - a new capability emerges
> - an existing spec's requirements are affected
>
> On confirmed upgrade: set `workflow: full`, `phase: design`,
> `metrics.upgraded: true`, **and annotate the proposal's first line to
> `Preset: tweak (upgraded to full YYYY-MM-DD)`** (the state-rebuild rules
> read that marker), then route through `/onto` to backfill the design
> phase.

## Exit checklist (per phase, lite)

- [ ] Open-lite: workspace exists, scope confirmed by the user
- [ ] Build: tasks checked + committed one by one, tree clean
- [ ] Verify: `verification.md` with fresh evidence + regression results
- [ ] Close: guides obligation resolved, confirmed, archived
