# Comet Design Handoff

- Change: onto-supersedes-edge
- Phase: design
- Mode: compact
- Context hash: 2ef365806d56b30f6536d90b84a4c1ab8c570eb9dc5bd9ceac8fe8c8001f99ad

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-supersedes-edge/proposal.md

- Source: openspec/changes/onto-supersedes-edge/proposal.md
- Lines: 1-35
- SHA256: 107b55811bd858e53949f5166138afeccee058f359fb6a0ba4c227ca4393fa49

```md
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

```

## openspec/changes/onto-supersedes-edge/design.md

- Source: openspec/changes/onto-supersedes-edge/design.md
- Lines: 1-31
- SHA256: 0ea93440b1a68b8846634a4a58f0bdfda7cc58ab5d6ae34c103bc395f1fc3a94

```md
# Design — onto supersedes edge

## State field

`ontostate.State` gains `Supersedes []string` (`yaml:"supersedes,omitempty"
json:"supersedes,omitempty"`), mirroring `Deps`. It is not gated (Validate
ignores it — B1: shape not judgment). `Save` round-trips it; only the set command
writes it.

## Set command

`supersedesCmd` mirrors `depsCmd`: `onto set supersedes <change> --change <name>
[--change …]` → `st.Supersedes = <names>` via `runTransition`. Registered in
`setCmd`. Repeatable `--change` (not a comma-split positional) so names carrying
edge characters are unambiguous.

## Graph edge

`buildGraph`'s per-change `add` emits, for each `st.Supersedes` entry, an edge
`{from: change, to: superseded, type: "supersedes"}` — after the depends-on and
implements edges. Deterministic ordering already sorts edges by (type, from, to).

## Test

- `onto set supersedes alpha --change old1 --change old2` → reload → Supersedes ==
  [old1 old2], other fields unchanged.
- `onto graph --json` over a change with `supersedes: [old]` → a supersedes edge
  change→old.

## Risk
Low — mirrors the deps field/setter and the graph edge pattern. Go tests pin it.

```

## openspec/changes/onto-supersedes-edge/tasks.md

- Source: openspec/changes/onto-supersedes-edge/tasks.md
- Lines: 1-10
- SHA256: 5e11a69d961ab390143a668fac245cd273d7a5c04a6a30e5e1cfd3a32c20f0f9

```md
# Tasks — onto-supersedes-edge

## 1. supersedes field + setter + graph edge
- [ ] State.Supersedes []string (ungated); `onto set supersedes <change>
      --change <name>...`; onto graph emits supersedes edges. TDD: setter
      round-trips; graph emits the supersedes edge.

## 2. Verify
- [ ] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet,
      build (incl cmd/onto), `openspec validate --all` green.

```

## openspec/changes/onto-supersedes-edge/specs/onto-binary/spec.md

- Source: openspec/changes/onto-supersedes-edge/specs/onto-binary/spec.md
- Lines: 1-37
- SHA256: 60565280c9b28401aa485f3181ad8949593bb5c323aa7e25564d295c1726fcc6

```md
# onto-binary

## MODIFIED Requirements

### Requirement: onto graph emits the change dependency traceability graph

`onto graph` SHALL emit the traceability graph over all onto changes, read-only
and config-independent. It MUST enumerate both active changes (`docs/changes/*`)
and archived changes (`docs/changes/archive/*`), emit one change node per change
(`kind: "change"`) carrying its stable id, name, phase, and archived flag (a
malformed or missing-state change still appears as a node labeled by its
directory, never silently dropped), and emit one `depends-on` edge for each entry
in a change's `deps`. It MUST also emit a capability node (`kind: "capability"`)
for each capability a change declares via a `specs/<capability>.md` delta-spec
file with an `implements` edge from the change to that capability, and a
`supersedes` edge from a change to each change named in its `supersedes` list.
With `--json` it MUST emit a `{nodes, edges}` object with deterministic ordering;
without it, a readable adjacency listing.

#### Scenario: graph lists dependency, implements, and supersedes edges

- **GIVEN** a change that depends on one change, implements a capability, and
  supersedes another change
- **WHEN** `onto graph` runs
- **THEN** it emits the `depends-on`, `implements`, and `supersedes` edges for that
  change

#### Scenario: onto set supersedes records the relationship

- **WHEN** `onto set supersedes <change> --change <name>` runs
- **THEN** the change's `onto-state.yaml` records `<name>` in its `supersedes`
  list, leaving other fields unchanged

#### Scenario: graph is read-only and needs no config

- **WHEN** `onto graph` runs in a workspace with no `homonto.toml`
- **THEN** it emits the graph without error and mutates no state

```
