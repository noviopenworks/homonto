---
comet_change: onto-stable-change-id
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-onto-stable-change-id
status: final
---

# onto-stable-change-id — Technical Design (X1)

OpenSpec is canonical; the delta records the requirement. X1's core for onto:
`onto-state.yaml` gains a stable `id` assigned once at `onto new` (crypto/rand
short hex), immutable across `set`/`advance`/`close`, surfaced in `state --json`/
`status`, empty (never retro-minted) for legacy states. onto is homonto's own
workflow, so this needs no change to the external comet/OpenSpec name-matching.

## Approach

`ontostate.State` gains `ID string yaml:"id,omitempty" json:"id,omitempty"`.
`onto new` sets it via a `newID()` (crypto/rand → 8 hex). No other writer sets
it; Save round-trips it, so set/advance/close preserve it. Load leaves an absent
id empty (no minting on read → an id never changes meaning across reads).

## Risk posture

Low — additive immutable field + generation at creation. crypto/rand in the Go
binary is fine (the no-random rule applies to comet workflow scripts, not this
binary). Go tests pin generation, uniqueness, and immutability across transitions.

## Out of scope

deps/refs by id (traceability-graph follow-on); retro-minting; the comet/OpenSpec
flow.
