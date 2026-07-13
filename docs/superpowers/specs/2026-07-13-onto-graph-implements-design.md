---
comet_change: onto-graph-implements
role: technical-design
canonical_spec: openspec
status: draft
---

# onto-graph-implements — Technical Design (X1)

OpenSpec is canonical; full approach in design.md. Extends `onto graph` with a
second typed edge: `graphNode` gains `Kind` ("change"|"capability"); a change's
`specs/<capability>.md` files yield capability nodes + `implements` edges
(change→capability). depends-on edges unchanged; read-only, deterministic. The
one further typed edge derivable from what onto records (tests/released-in/
supersedes need untracked data — a separate design decision).

## Risk posture

Low — a read-only ReadDir of each change's specs dir added to the enumerator.
Tests pin the capability node + implements edge + JSON kind field.

## Out of scope

tests/released-in/supersedes/deviates-from edges; CI validation.
