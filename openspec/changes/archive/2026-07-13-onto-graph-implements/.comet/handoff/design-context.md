# Comet Design Handoff

- Change: onto-graph-implements
- Phase: design
- Mode: compact
- Context hash: ce0aa62388dcaabbe08be2cb15b926c40242b64a6e0455fa780c0c29b48ea80c

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-graph-implements/proposal.md

- Source: openspec/changes/onto-graph-implements/proposal.md
- Lines: 1-35
- SHA256: 447fed856d69dccc27093e43b9a6c0656828e36263941ad24ed03f9ec3654261

```md
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

```

## openspec/changes/onto-graph-implements/design.md

- Source: openspec/changes/onto-graph-implements/design.md
- Lines: 1-33
- SHA256: b71ac89f538a98003ea945124cac99ae9d5ca636a1943906dc36466fcbf8b0e0

```md
# Design — onto graph implements edges

## Node kinds

`graphNode` gains `Kind string` (`"change"` | `"capability"`). Change nodes set
`kind: "change"` (id/phase/archived as today); capability nodes are
`{kind: "capability", change: <capability>}` (id/phase empty), deduplicated by
name across all changes.

## implements edges

For each change directory, read `<dir>/specs/` for `*.md` files (onto's delta-
spec layout is `specs/<capability>.md`). Each file names a capability
(`<capability>` = filename without `.md`); emit an `implements` edge
`{from: change, to: capability, type: "implements"}` and ensure a capability
node exists. A change with no `specs/` dir (or an empty one) contributes nothing
— unchanged.

## Output

Nodes now carry `kind`; edges include both `depends-on` and `implements`.
Deterministic: nodes sorted by (kind, name), edges by (type, from, to). `--json`
shape is `{nodes:[{id,change,phase,archived,kind}], edges:[{from,to,type}]}`.

## Risk

Low — read-only `os.ReadDir` of each change's specs dir added to the existing
enumerator. Tests build a change with a `specs/<cap>.md` and assert the
capability node + implements edge + JSON.

## Alternatives
- Emit implements edges to spec *files* rather than capability names — rejected;
  the capability (the file's basename) is the stable traceability target.

```

## openspec/changes/onto-graph-implements/tasks.md

- Source: openspec/changes/onto-graph-implements/tasks.md
- Lines: 1-11
- SHA256: d30724f9d66278239785d4f91a69586cb709eb458423f06b7da295b6168035fa

```md
# Tasks — onto-graph-implements

## 1. Capability nodes + implements edges
- [ ] onto graph emits capability nodes (kind) and implements edges (change ->
      capability from specs/<cap>.md); change nodes get kind:"change". Read-only,
      deterministic. TDD: a change with a specs/<cap>.md yields the capability
      node + implements edge; JSON shape carries kind.

## 2. Verify
- [ ] `go test ./internal/ontocli/... -race`, vet, build (incl cmd/onto),
      `openspec validate --all` green.

```

## openspec/changes/onto-graph-implements/specs/onto-binary/spec.md

- Source: openspec/changes/onto-graph-implements/specs/onto-binary/spec.md
- Lines: 1-36
- SHA256: b02a89e575c048644d5f19c4d448436d7e59f5fc14301d19bd89e873e903601c

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
file, and an `implements` edge from the change to that capability. With `--json`
it MUST emit a `{nodes, edges}` object with deterministic ordering; without it, a
readable adjacency listing.

#### Scenario: graph lists active and archived changes with their dependencies

- **GIVEN** an active change depending on an archived change
- **WHEN** `onto graph` runs
- **THEN** both appear as change nodes (with id/phase/archived) and a `depends-on`
  edge links the dependent to its dependency

#### Scenario: graph lists implemented capabilities

- **GIVEN** a change with a `specs/<capability>.md` delta spec
- **WHEN** `onto graph` runs
- **THEN** the capability appears as a node and an `implements` edge links the
  change to it

#### Scenario: graph is read-only and needs no config

- **WHEN** `onto graph` runs in a workspace with no `homonto.toml`
- **THEN** it emits the graph without error and mutates no state

```
