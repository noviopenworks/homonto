# onto graph: add capability nodes and implements edges

## Why

Roadmap X1, extending the traceability graph (`onto-graph-command`) with a second
typed edge. A change's delta specs (`specs/<capability>.md`) record which
capabilities it modifies — the `implements` relationship. Surfacing it answers
"which changes touch capability X" and moves onto's graph from
changes-and-dependencies toward the typed traceability graph X1 calls for. This
is the one further edge type derivable from what onto already records; the rest
(`tests`/`released-in`/`supersedes`) would need data onto does not yet track — a
separate design decision, not a mechanical add.

## What Changes

- `onto graph` gains **capability nodes** (`kind: "capability"`) and
  **`implements` edges** (change → each capability named by a
  `specs/<capability>.md` file in the change directory). Existing change nodes
  gain `kind: "change"`; `depends-on` edges are unchanged.
- Read-only, config-independent, deterministic ordering — unchanged from the
  existing command.

## Impact

- **Specs:** the `onto-binary` "onto graph" requirement is extended (MODIFIED) to
  include capability nodes and implements edges.
- **Behavior:** additive to the existing command; a change with no `specs/` dir
  contributes no capability nodes/edges (unchanged output for such changes).
- **Risk:** low — a read-only enumerator extension; Go tests pin the capability
  nodes, implements edges, and JSON shape.

## Non-goals

- `tests`/`released-in`/`supersedes`/`deviates-from` edges (onto does not track
  the linking data — a design decision on what to record); CI validation.
