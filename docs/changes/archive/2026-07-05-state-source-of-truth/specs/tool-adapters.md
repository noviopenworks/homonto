# Delta Spec: tool-adapters (state-source-of-truth)

## ADDED Requirements

### Requirement: Adapters adopt pre-existing matching keys

Each adapter SHALL, on apply, record in state a declared non-secret key that is
present on disk, equal to its desired value, and absent from state — rather than
leaving it untracked — so that pruning and drift detection both see it. The
claude and opencode adapters SHALL behave identically in this respect, including
opencode plugins recorded by array membership. Adoption SHALL NOT modify the
tool file (the on-disk value already matches desired) and SHALL never apply to
secret-bearing keys.

#### Scenario: Claude adopts a pre-existing MCP

- **GIVEN** an MCP declared for claude whose `~/.claude.json` entry already
  equals the desired projection and which is absent from state
- **WHEN** apply runs
- **THEN** state gains an `mcp.<name>` record for claude, `~/.claude.json` is
  left byte-unchanged, and a later removal of that MCP from config prunes it

#### Scenario: OpenCode adopts a pre-existing setting and plugin

- **GIVEN** an opencode setting and an opencode plugin already present in
  `opencode.jsonc` matching desired, both absent from state
- **WHEN** apply runs
- **THEN** state gains `setting.<key>` and `plugin.<name>` records for opencode,
  `opencode.jsonc` is left byte-unchanged (its comments preserved, because
  adoption writes no tool file), and both become pruneable on later removal
  from config
