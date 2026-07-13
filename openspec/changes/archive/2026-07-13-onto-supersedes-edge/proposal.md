# onto: a supersedes relationship and its traceability edge

## Why

Roadmap X1, the next typed traceability edge. `depends-on` and `implements` are
delivered; `supersedes` (a change that replaces/obsoletes an earlier change) is a
real relationship the graph should carry, but onto did not record it. **Schema
decision (made):** `supersedes` is a list of change names — exactly like `deps` —
declared on the change that does the superseding, settable through the existing
`onto set` machinery. With the field recorded, `onto graph` derives the edge.

## What Changes

- `onto-state.yaml` gains `supersedes` (a `[]string` of change names), settable
  via `onto set supersedes <change> --change <name> [--change …]` (mirroring
  `onto set deps --dep`). Absent/empty by default; legacy states are unchanged.
- `onto graph` emits a `supersedes` edge (change → each superseded change) for
  every `supersedes` entry, alongside the existing `depends-on` and `implements`
  edges.

## Impact

- **Specs:** the `onto-binary` "onto graph" requirement is extended (MODIFIED) to
  include `supersedes` edges; a note records the new state field.
- **Behavior:** additive; a change with no `supersedes` behaves as before.
- **Risk:** low — a new list field + a set subcommand mirroring `deps` + one more
  derived edge; Go tests pin the setter, immutability of unrelated fields, and the
  graph edge.

## Non-goals

- `tests`/`released-in`/`deviates-from` edges — those need data onto still does
  not track (test-coverage, release, deviation), separate design decisions.
- Validating that a superseded change exists (a superseded change may already be
  archived or removed; the edge records the declared relationship).
