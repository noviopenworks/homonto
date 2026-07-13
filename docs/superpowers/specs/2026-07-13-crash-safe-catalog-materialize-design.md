---
comet_change: crash-safe-catalog-materialize
role: technical-design
canonical_spec: openspec
status: draft
---

# crash-safe-catalog-materialize — Technical Design

Deep design for X2's F47 catalog-materialization slice. OpenSpec is the
canonical spec; this defers to `openspec/changes/crash-safe-catalog-materialize/
specs` for normative requirements; the full approach is in the change's
`design.md`.

## Decision

Make `catalog.Materialize` atomic per skill via stage-then-swap: write each
skill's files into a `<dst>.staging` sibling, then `RemoveAll(dst)` +
`Rename(staging, dst)` only after the full walk succeeds. A mid-walk failure
leaves the prior complete `dst` untouched; a crash in the swap window leaves
`dst` absent (re-materialized next run), never partial.

## Why this slice

`allSkillDirsExist` only Stats the destination, so a partial skill dir (from a
crash between `RemoveAll` and completing the walk today) passes the gate forever
and is never repaired. The atomic swap restores directory-presence as a valid
completeness signal. Commands/subagents already write atomically (single-file
WriteControlPlane), so only the multi-file skill directory needs this.

## Risk posture

Low — localized to one function, no success-path byte change, guarded by the
catalog + engine materialize suites plus a new failure-path test.

## Out of scope

Commands/subagents staging (already atomic); a completion-marker gate; broader
X2 (stateless Apply, transaction journals, close/archive validation).
