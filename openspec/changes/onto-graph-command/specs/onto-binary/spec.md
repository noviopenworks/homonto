# onto-binary

## ADDED Requirements

### Requirement: onto graph emits the change dependency traceability graph

`onto graph` SHALL emit the dependency graph over all onto changes, read-only and
config-independent. It MUST enumerate both active changes (`docs/changes/*`) and
archived changes (`docs/changes/archive/*`), emit one node per change carrying its
stable id, name, phase, and archived flag (a malformed or missing-state change
still appears as a node labeled by its directory, never silently dropped), and
emit one `depends-on` edge for each entry in a change's `deps`. With `--json` it
MUST emit a `{nodes, edges}` object with deterministic ordering; without it, a
readable adjacency listing.

#### Scenario: graph lists active and archived changes with their dependencies

- **GIVEN** an active change depending on an archived change
- **WHEN** `onto graph` runs
- **THEN** both appear as nodes (with id/phase/archived) and a `depends-on` edge
  links the dependent to its dependency

#### Scenario: graph is read-only and needs no config

- **WHEN** `onto graph` runs in a workspace with no `homonto.toml`
- **THEN** it emits the graph without error and mutates no state
