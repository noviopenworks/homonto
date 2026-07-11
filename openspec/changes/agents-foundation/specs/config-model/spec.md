## ADDED Requirements

### Requirement: Agent lifecycle declaration

Lifecycle-managed agents SHALL be declarable as `[agents.<name>]` tables, distinct
from the v1 `[subagents.<name>]` symlink model. Each agent table SHALL carry:

- `source` (required): the agent source, using the `builtin:<name>` or
  `local:<name>` scheme (remote schemes are not yet accepted);
- `version` (optional string): a pinned version; empty means unpinned;
- `targets` (optional list): target tools ∈ {`claude`, `opencode`}; empty means
  both;
- `mode` (optional): `copy` or `link`; empty defaults to `link`.

The agent name SHALL be validated as a config key. An invalid source scheme, an
unknown target, or an invalid `mode` SHALL be rejected at load naming the agent.

#### Scenario: Parse an agent declaration

- **GIVEN** `[agents.review]` with `source = "builtin:review-agent"`, `version = "1.2.0"`, `targets = ["claude","opencode"]`, `mode = "copy"`
- **WHEN** the config is parsed
- **THEN** it yields an agent `review` with that source, version, targets, and mode

#### Scenario: Defaults for optional fields

- **GIVEN** `[agents.x]` with only `source = "local:x"`
- **WHEN** the config is parsed
- **THEN** the agent has empty version (unpinned), both tools as targets, and mode `link`

#### Scenario: Invalid agent source is rejected

- **GIVEN** `[agents.x]` with `source = "https://example.com/x"`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the agent and the invalid source

#### Scenario: Invalid agent mode is rejected

- **GIVEN** `[agents.x]` with `source = "builtin:x"` and `mode = "symlink"`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the agent and the invalid mode
