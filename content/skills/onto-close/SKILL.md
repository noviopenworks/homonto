---
name: onto-close
description: onto phase 5 — close. Use when an active change has phase close (verification passed) — merges spec deltas, numbers and accepts ADR drafts, enforces the guides obligation, and archives the workspace after final confirmation.
---

# onto-close — Phase 5: Close

Land the change's knowledge where it lives permanently: living specs, the
ADR log, and user-facing guides — then archive the workspace.

## Entry check

- `state.yaml` has `phase: close`; `verification.md` exists with a
  `Result: pass` line (accepted deviations, if any, are recorded inside the
  report; the result enum stays `pass`).
- Read `notes.md` at entry when present — recorded directives determine
  whether the final gate is pre-authorized.
- Execute any `DEFERRED to close:` tasks from `tasks.md` during this phase
  (before the final confirmation); the pre-archive lint blocks on
  unresolved markers.
- Anything else → route back through `/onto`.

## Steps

### 0. Lint (blocking)

Run `references/lint-checklist.md` sections 1–2 (delta format, workspace
state) now; section 3 (post-merge) after step 1; section 4 (guides
resolved + dangling references) before archiving. **Findings block the
archive exactly like the guides obligation** — fix or stop. This replaces
the format validation the retired external tooling used to perform.

### 1. Merge spec deltas

For each workspace delta `specs/<capability>.md`, merge into
`docs/specs/<capability>.md` per the semantics in `docs/specs/README.md`,
applying sections in this order — **RENAMED first, then MODIFIED, then
REMOVED, then ADDED** (so a MODIFIED block targeting a RENAMED new name
finds it):

- `## RENAMED Requirements` → rename the heading per each FROM/TO pair,
  preserving the body
- `## MODIFIED Requirements` → replace the requirement of the same name
  (which may be a just-renamed name)
- `## REMOVED Requirements` → delete the named requirement
- `## ADDED Requirements` → append the requirement blocks
- Delta for a capability with no living spec → create the file (strip the
  ADDED wrapper; the living spec has plain `## Requirements`)

The merged living spec must read as "always true, now" — no change-log
language.

### 2. Number and accept ADRs

For each draft in the workspace `adr/`:

1. Next free number = highest `NNNN` in `docs/adr/` + 1.
2. Move the draft to `docs/adr/NNNN-<slug>.md` (git mv).
3. Set `Status: Accepted`. If it supersedes an existing ADR, set that one's
   status to `Superseded by NNNN`.

### 3. Guides obligation (hard block)

Check `guides:` in `state.yaml`. Archiving with `guides: pending` is
**prohibited**:

- Write or update the affected `docs/guides/<topic>.md` (and README if the
  change is user-visible) → set `guides: updated`, **or**
- Record `guides: "waived: <reason>"` (quoted — a bare `waived: <reason>`
  is invalid YAML) — the reason must come from the user or a recorded
  directive, never invented.

### 4. Final confirmation

> **GATE (final confirmation):** summarize what was merged (specs), numbered
> (ADRs), and documented (guides), then ask for confirmation to archive.
> This gate MAY be pre-authorized by a verbatim recorded directive — still
> surface the summary.

### 5. Finalize metrics and archive

1. Finalize `metrics` in `state.yaml`: `phases.close: <today>`,
   `tasks_total` (checked tasks), `verify_rounds`, `upgraded`. Metrics are
   observational — never block on them.
2. `git mv docs/changes/<name> docs/changes/archive/YYYY-MM-DD-<name>`
   (today's date).
3. Set `archived: true` in the moved `state.yaml`. `phase` intentionally
   stays `close` — "done" is a derived-only phase, never written.
4. Commit. The archived workspace is history — never edited afterwards,
   with exactly one sanctioned exception: `ship.md` (step 6).

### 6. Ship handoff (offer)

Follow `references/ship-handoff.md`: offer the ready PR body assembled
from the archived change; if accepted, write it to the archive's
`ship.md` and name the PR skills as the next step. onto itself never
pushes or opens PRs.

## Exit checklist

- [ ] Lint checklist fully passed (pre-merge, post-merge, dangling refs)
- [ ] Every delta spec merged (incl. RENAMED); living specs read as
      current truth
- [ ] Every ADR draft numbered, accepted, moved to `docs/adr/`
- [ ] `guides: updated` or `guides: "waived: <reason>"` — never pending
- [ ] Metrics finalized (phase dates, tasks_total, verify_rounds, upgraded)
- [ ] Final confirmation given (or pre-authorized verbatim directive)
- [ ] Workspace under `docs/changes/archive/YYYY-MM-DD-<name>/` with
      `archived: true`, everything committed
- [ ] Ship handoff offered (ship.md written if accepted)
- [ ] Announce completion and summarize where the knowledge landed
