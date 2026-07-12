---
name: onto-close
description: onto phase 5 — close. Use when an active change has phase close (verification passed) — merges spec deltas, numbers and accepts ADR drafts, enforces the guides obligation, and archives the workspace after final confirmation.
---

# onto-close — Phase 5: Close

Land the change's knowledge where it lives permanently: living specs, the
ADR log, and user-facing guides — then archive the workspace.

## Entry check

- `state.yaml` has `phase: close`; `verification.md` exists with a
  `Result: pass` line (a trailing `(N accepted deviations)` still counts —
  match the `Result: pass` prefix; the deviations are recorded inside the
  report).
- **Idempotent re-entry**: close mutates shared files (living specs, the
  ADR log). If `state.yaml` shows `close.merged: true`, the deltas and
  ADRs already landed on a prior, interrupted close — do NOT merge or
  number again (a second ADDED merge duplicates requirements). Resume at
  the guides/gate/archive steps. The merge step sets `close.merged: true`
  before it starts and records what it did, so a re-entry knows to skip it.
- Read `notes.md` at entry when present — the final gate is pre-authorized
  only by a directive that explicitly names closing/archiving (step 2).
- Anything else → route back through `/onto`.

## Steps

Close mutates shared, permanent files (living specs, the ADR log). So the
order is deliberate: **prepare and confirm first, mutate second, archive
atomically** — no global change happens before the final gate, and the
one interruption-prone step (mv + archived flag) is a single commit.

### 1. Lint and prepare (blocking, no global mutation yet)

- Run `references/lint-checklist.md` sections 0–2 (delta coverage, delta
  format, workspace state). Section 0 is the one that catches a behavior
  change shipping with no spec. Findings block close — fix or stop. This
  replaces the format validation the retired external tooling performed.
- Execute any `DEFERRED to close:` tasks from `tasks.md` now (they must be
  non-runtime — bookkeeping, file moves, doc stamps — because verify never
  exercised them). Rewrite each executed line to
  `- [x] N.N (deferred, done at close YYYY-MM-DD): <desc>` and note the
  evidence. If executing one turns out to change runtime behavior, **stop**:
  it should have been built before verify. Route back to build, add a task,
  re-verify — closing unverified runtime behavior is exactly the hole the
  deferral rule exists to prevent.
- Resolve the **guides obligation** (`guides:` in `state.yaml`); archiving
  with `guides: pending` is prohibited. Either write/update the affected
  `docs/guides/<topic>.md` (and README if user-visible) → `guides: updated`,
  **or** record `guides: "waived: <reason>"` (quoted — a bare
  `waived: <reason>` is invalid YAML; the reason comes from the user or a
  recorded directive, never invented). Guide prose gets the onto-no-slop
  pass; the specs and ADRs do not yet exist in living form, so they wait
  for step 3.
- Assemble the **close plan**: each workspace delta → its target
  `docs/specs/<capability>.md` and the operations it applies; each ADR
  draft → its next number and slug; the guides outcome; the deferred tasks
  executed. This plan is what the gate shows.

### 2. Final confirmation gate (before any spec or ADR mutation)

> **GATE (final confirmation):** present the close plan — deltas to merge
> and how, ADR numbers + slugs to assign, guides outcome, deferred tasks
> done — and ask for confirmation to merge and archive. Nothing global has
> changed yet, so a declined gate leaves the repo untouched.
>
> This gate MAY be pre-authorized **only** by a recorded directive that
> explicitly authorizes closing/archiving this change (e.g. "close and
> archive it when done"). A generic build directive like "run to
> completion with defaults" does **not** reach this gate — archiving is
> irreversible; a directive must name it. Still surface the plan.

### 3. Execute the close (only after confirmation)

1. Set `close.merged: true` in `state.yaml` **before merging**, so an
   interruption mid-merge is recognized on re-entry and the merge is not
   re-run (a second ADDED merge would duplicate requirements).
2. **Merge spec deltas.** For each workspace delta `specs/<capability>.md`,
   merge into `docs/specs/<capability>.md` by the semantics below (the same
   ones `references/specs-readme.md` records for the repo's
   `docs/specs/README.md`), applying sections **RENAMED first, then
   MODIFIED, then REMOVED, then ADDED** (so a MODIFIED block targeting a
   renamed name finds it):
   - `## RENAMED Requirements` → rename the heading per each FROM/TO pair,
     preserving the body
   - `## MODIFIED Requirements` → replace the requirement of the same name
     (which may be a just-renamed name) — replace it *entirely*, do not
     append; a MODIFIED that leaves the old block in place is a defect the
     post-merge lint's duplicate check catches
   - `## REMOVED Requirements` → delete the named requirement
   - `## ADDED Requirements` → append the requirement blocks
   - Delta for a capability with no living spec → create the file (strip
     the ADDED wrapper; the living spec has plain `## Requirements`)

   The merged spec reads as "always true, now" — no change-log language.
   **onto-no-slop applies to genuinely new prose only** — never rewrite a
   requirement's normative wording, never touch a `SHALL`/`MUST` line, a
   scenario's GIVEN/WHEN/THEN, or any machine-read marker.
3. **Number and accept ADRs.** For each draft in the workspace `adr/`:
   next free number = highest `NNNN` in `docs/adr/` + 1; `git mv` to
   `docs/adr/NNNN-<slug>.md`; set `Status: Accepted` (and any superseded
   ADR → `Superseded by NNNN`). Assign numbers to all drafts in one pass
   before moving any, so two drafts never collide on the same number.
   Then rewrite the workspace's `design.md` and `notes.md` references from
   `adr/<slug>.md` to the final `docs/adr/NNNN-<slug>.md` path — otherwise
   the archive ships dangling ADR references.
4. Run lint-checklist section 3 (post-merge: no delta-only headings leaked,
   no duplicated requirements, scenario structure intact) and section 4
   (guides resolved, no dangling references). Findings block the archive.
5. Finalize `metrics`: `phases.close: <today>`, `tasks_total`,
   `verify_rounds`, `upgraded`. Observational — never block on them.
6. **Archive atomically**: `git mv docs/changes/<name>
   docs/changes/archive/YYYY-MM-DD-<name>`, set `archived: true` in the
   moved `state.yaml`, and **commit both in one commit** — no window where
   a workspace sits under `archive/` still reading `archived: false`
   (discovery excludes `archive/`, so such a change would be invisible).
   `phase` stays `close`; "done" is derived-only, never written. The
   archived workspace is history — never edited after, with one sanctioned
   exception: `ship.md`.

### 4. Ship handoff (offer)

Follow `references/ship-handoff.md`: offer the ready PR body assembled
from the archived change; if accepted, write it to the archive's
`ship.md` and name the PR skills as the next step. onto itself never
pushes or opens PRs.

## Exit checklist

- [ ] Final confirmation given **before** any spec/ADR mutation (or a
      directive that explicitly authorized closing/archiving)
- [ ] `close.merged: true` set before the merge (idempotency guard)
- [ ] Lint checklist fully passed (pre-merge §1–2, post-merge §3 incl. the
      duplicate-requirement check, pre-archive §4 dangling refs)
- [ ] Every delta spec merged (RENAMED→MODIFIED→REMOVED→ADDED); living
      specs read as current truth with no duplicated requirements
- [ ] Every ADR draft numbered, accepted, moved to `docs/adr/`; workspace
      references rewritten to the final paths
- [ ] `guides: updated` or `guides: "waived: <reason>"` — never pending
- [ ] onto-no-slop pass run over **new** guide/ADR prose only, scores in
      `notes.md`; no requirement wording, `SHALL`/`MUST` line, scenario, or
      machine-read marker was rewritten
- [ ] Metrics finalized (phase dates, tasks_total, verify_rounds, upgraded)
- [ ] Archive is one commit: workspace under
      `docs/changes/archive/YYYY-MM-DD-<name>/` **and** `archived: true`,
      committed together, everything tracked
- [ ] Ship handoff offered (ship.md written if accepted)
- [ ] Announce completion and summarize where the knowledge landed
