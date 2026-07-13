---
comet_change: config-load-phases
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-config-load-phases
status: final
---

# config-load-phases — Technical Design

Deep design for X3's F43 config-monolith slice. OpenSpec is the canonical spec;
this defers to `openspec/changes/config-load-phases/specs` for normative
requirements; the full approach is in the change's `design.md`.

## Decision

Split the ~200-line `config.Load` into explicit ordered phase functions —
`decode` (parse + schema-version guard), `migrate` (`[agents]`→`[subagents]`
fold), `normalize` (scope defaulting), `validate` (the whole validation block) —
extracted verbatim and in the same order. `Load` becomes read → decode →
migrate → normalize → validate.

## Why

The X3 exit gate wants config loading to run as explicit phases "ending the
monolith". This slice does exactly that structurally, without touching any
validation rule, so it is a pure legibility/testability win pinned by the
existing comprehensive config suite.

## Risk posture

Low — mechanical in-order extraction, no reordering, no rule change. Every config
load/validation test is the safety net; any behavior diff means the extraction
slipped.

## Out of scope

The generic per-kind "expand" pipeline (a larger follow-on); any validation
change.
