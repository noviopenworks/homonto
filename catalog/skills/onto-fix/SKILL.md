---
name: onto-fix
description: onto preset — bug fix. Use for behavior fixes that need no new capability design — open-lite, then build starting from a failing test that reproduces the bug, verify, close; upgrades to the full workflow when scope grows.
---

# onto-fix — Preset: Bug Fix

Fast path for fixing broken behavior: **open-lite → build → verify → close**.
Skips the design phase — which is exactly why the upgrade rules below are
non-negotiable.

## Entry check

- A new bug-fix request (clear broken behavior), or an active change with
  `workflow: fix`. This preset owns the change's whole lifecycle; the
  dispatcher routes every phase of a fix change here.
- Not for new capabilities, refactors, or behavior *changes* — those are
  full-workflow work via `onto-open`.
- Read `notes.md` at entry when present. If any skill's `references/`
  directory is missing, degrade per the dispatcher rule: reconstruct from
  the `docs/` contract pointers, note the gap, continue.

## Steps

### 1. Open-lite

Minimal clarification: reproduction steps, expected vs actual behavior,
suspected blast radius. Create `docs/changes/<name>/` with:

- `state.yaml` — `workflow: fix`, `phase: build` (no design phase),
  `created`, `base_ref`, `guides: pending`, and `decisions` defaulted at
  open-lite since presets enter build directly: `isolation: branch`,
  `execution: direct`, `tdd: direct` (the failing-test-first rule below is
  independent of the `tdd` field); rest per `docs/changes/README.md`
- `proposal.md` — first line `Preset: fix` (the dispatcher's state rebuild
  keys on this marker), then the bug (link the issue if there is one),
  reproduction, expected behavior, fix scope
- `tasks.md` — short checklist (reproduce → fix → regression)

No full design and no plan.md required. Branch: `fix/YYYYMMDD/<name>`.
Templates: reuse the full-workflow references (`onto/references/state-yaml.md`,
`onto-open/references/{proposal,tasks,notes}.md`) — a `notes.md` checkpoint
is recommended for any fix that takes more than one sitting. Stamp
`metrics.phases.<phase>` at each phase exit like the full workflow.

### 2. Build — failing test first, always

**A failing test that reproduces the bug is required FIRST, regardless of
the `tdd` decision.** Watch it fail for the expected reason. Then find the
root cause (systematic debugging — reproduce, read the whole error, trace
data flow; no fix before the root cause is identified), apply the minimal
fix, watch the test pass, run the surrounding tests. One commit per task.

### 3. Verify (light)

`verify.mode: light` unless upgraded. The bug's reproduction is the core
scenario: demonstrate it no longer occurs, with the literal command +
output in `docs/changes/<name>/verification.md` (template:
`onto-verify/references/verification.md`), plus regression-suite results.
One adversarial skeptic is optional in light mode (protocol:
`onto-verify/references/adversarial.md`); record a skip. Failure → same
gate as the full workflow (fix or accept-deviation, fresh user input).

### 4. Close

Same obligations as `onto-close` — lint (`onto-close/references/
lint-checklist.md`), spec deltas merged if any requirement changed, guides
checked (`updated` or `"waived: <reason>"`), metrics finalized, final
confirmation, archive to `docs/changes/archive/YYYY-MM-DD-<name>/`, ship
handoff offered.

## Upgrade rules

> **GATE (upgrade):** the moment ANY of these becomes true, pause, explain
> the trigger, and require fresh user confirmation to upgrade:
>
> - the fix touches **more than 5 non-test files** (the mandatory failing
>   test never counts toward the trigger; aligned with tweak's limit so a
>   fix never carries more ceremony than a same-sized feature)
> - architecture or schema changes (new modules, interfaces, dependencies)
> - the fix introduces a **new public API**
> - the fix scope exceeds a single function/module
>
> On confirmed upgrade: set `workflow: full`, `phase: design`,
> `metrics.upgraded: true` in `state.yaml`, **and annotate the proposal's
> first line to `Preset: fix (upgraded to full YYYY-MM-DD)`** — the
> state-rebuild rules read that marker, so an upgrade must survive state
> loss. Then route through `/onto` to backfill the design phase. Never
> keep patching past a trigger "because it's almost done".

## Exit checklist (per phase, lite)

- [ ] Open-lite: workspace + reproduction confirmed by the user
- [ ] Build: failing test seen failing, root cause stated, fix committed,
      test seen passing, tree clean
- [ ] Verify: `verification.md` with reproduction evidence + regression
      results; `verify.result` set
- [ ] Close: guides obligation resolved, confirmed, archived
- [ ] onto-no-slop pass run over every prose artifact (proposal, verification,
      guides, commit messages)
