# onto-binary

## ADDED Requirements

### Requirement: onto graph detects change-dependency cycles

`onto graph` SHALL detect cycles in the change-dependency (`depends-on`) graph and
report them deterministically. A cycle is a sequence of changes each depending on
the next, closing back on itself. Detection MUST consider only `depends-on` edges,
MUST be order-independent (the same set of cycles regardless of directory read
order), and MUST report each cycle as an ordered list of change names.

With `--json`, `onto graph` MUST include a `cycles` array (empty when the graph is
acyclic) alongside `nodes` and `edges`. In the human listing, detected cycles MUST
appear in a trailing `cycles:` section; an acyclic graph adds nothing.

`onto graph --check` MUST exit non-zero when at least one dependency cycle exists,
reporting the offending cycle(s), and exit zero when the graph is acyclic. Without
`--check`, `onto graph` reports cycles but still exits zero (it remains a read-only
inspection command).

#### Scenario: graph reports a dependency cycle

- **GIVEN** changes `a` (deps: `b`) and `b` (deps: `a`)
- **WHEN** `onto graph --json` runs
- **THEN** its `cycles` array contains a cycle listing both `a` and `b`

#### Scenario: --check fails on a cycle

- **GIVEN** a change graph containing a dependency cycle
- **WHEN** `onto graph --check` runs
- **THEN** it exits non-zero and reports the cycle

#### Scenario: --check passes on an acyclic graph

- **GIVEN** a change graph with no dependency cycle
- **WHEN** `onto graph --check` runs
- **THEN** it exits zero
