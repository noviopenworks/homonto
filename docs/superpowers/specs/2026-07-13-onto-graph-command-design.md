---
comet_change: onto-graph-command
role: technical-design
canonical_spec: openspec
status: draft
---

# onto-graph-command — Technical Design (X1)

OpenSpec is canonical; full approach in the change's design.md. `onto graph
[--json]` is a read-only, config-independent enumerator (mirroring `onto status`)
that emits the change dependency graph: nodes (stable id, name, phase, archived)
over active + archived changes, and `depends-on` edges from each change's deps.
Builds on the stable-id core; the first slice of X1's traceability graph.

## Risk posture

Low — read-only, reuses the status classification (F14: a malformed/missing-state
change still appears as a node). Go tests pin nodes/edges/JSON over a few changes.

## Out of scope

The full typed-edge set (implements/tests/supersedes/deviates-from/released-in);
edges by id (deps are name-keyed today); CI validation; the comet/OpenSpec flow.
