---
comet_change: typed-plan-operations
role: technical-design
canonical_spec: openspec
status: draft
---

# typed-plan-operations — Technical Design

Deep design for X2's typed-operations slice (F41). OpenSpec is the canonical
spec; this is the technical design and defers to `openspec/changes/
typed-plan-operations/specs` for normative requirements; the full API and
engine wiring are in the change's `design.md`.

## Decision

Refine `adapter.Change.Action` from a bare `string` (enum-in-a-comment) to a
defined `adapter.Action` type with exported constants and a `Valid()` method,
and add `ChangeSet.Validate(knownTools)`. `engine.Apply` calls it fail-closed
for every set before any secret resolution, materialization, or write.

## Why this is the right first X2 slice

X2 is large (typed immutable plans + transaction journals + staging + close
validation). The typed-and-validated-operations piece is the smallest
self-contained safety win: it closes a real fail-open gap (an unknown-tool set
is silently skipped today; an unknown action silently no-ops) without the much
larger statelessness/journal refactors. It is low-churn because the typed
constants keep the same underlying string values, so existing construction and
comparison sites keep compiling.

## Risk posture

Low. Additive validation on a fail-closed path plus a non-breaking type
refinement. No legal plan changes behavior — every action a real adapter emits
is valid and every set's tool is registered. The only observable change is that
a previously-silent unknown-tool/action set now aborts with a clear error.

## Out of scope (later X2 slices)

Stateless Apply (not reading adapter fields set by Plan); transaction journals;
versioned staging trees; close/archive validation (F42/F47/F4/F18); typing the
Key/Old/New payloads.
