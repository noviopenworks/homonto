---
comet_change: close-archive-rollback
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-close-archive-rollback
status: final
---

# close-archive-rollback — Technical Design

Deep design for X2's F4 onto-close-consistency slice. OpenSpec is the canonical
spec; this defers to `openspec/changes/close-archive-rollback/specs` for
normative requirements; the full approach is in the change's `design.md`.

## Decision

In `internal/ontocli/close.go` `runClose`, wrap the destructive tail so a
failure after writing `archived: true` rolls the flag back to `false` and
re-saves the in-place `onto-state.yaml`. A failed archive move now leaves the
change fully un-archived (spec: "archives NOTHING" on any failing case), instead
of the current stale `archived: true` at the original path.

## Why save-then-rollback

The `archived` flag lives in `onto-state.yaml` inside the change dir, so saving
it before the atomic rename keeps the on-success record co-located (the rename
carries it into the archive with no second write). The added cost is the
error-path rollback. Deriving archived-ness from directory location would close
the crash window entirely but is a larger redesign — out of scope; this fixes
the deterministic error path.

## Risk posture

Low — a localized error-path rollback, no success-path change, covered by a new
rename-failure-injection test plus the existing onto close suite.

## Out of scope

Full crash-safety (a kill between save and rename still has a window);
location-derived archived state; broader X2.
