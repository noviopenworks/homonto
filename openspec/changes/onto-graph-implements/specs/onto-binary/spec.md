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
