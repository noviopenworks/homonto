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
file with an `implements` edge from the change to that capability, a
`supersedes` edge from a change to each change named in its `supersedes` list,
and a `deviates-from` edge from a change to each target named in its
`deviates-from` list. With `--json` it MUST emit a `{nodes, edges}` object with
deterministic ordering; without it, a readable adjacency listing.

#### Scenario: graph lists dependency, implements, supersedes, and deviates-from edges

- **GIVEN** a change that depends on one change, implements a capability,
  supersedes another change, and deviates from a decision
- **WHEN** `onto graph` runs
- **THEN** it emits the `depends-on`, `implements`, `supersedes`, and
  `deviates-from` edges for that change

#### Scenario: onto set supersedes records the relationship

- **WHEN** `onto set supersedes <change> --change <name>` runs
- **THEN** the change's `onto-state.yaml` records `<name>` in its `supersedes`
  list, leaving other fields unchanged

#### Scenario: onto set deviates-from records the relationship

- **WHEN** `onto set deviates-from <change> --from <name>` runs
- **THEN** the change's `onto-state.yaml` records `<name>` in its `deviates-from`
  list, leaving other fields unchanged

#### Scenario: graph is read-only and needs no config

- **WHEN** `onto graph` runs in a workspace with no `homonto.toml`
- **THEN** it emits the graph without error and mutates no state
