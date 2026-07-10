## MODIFIED Requirements

### Requirement: Local provider content root

Local provider content SHALL live under `homonto/` relative to the directory containing `homonto.toml`; generated state, cache, and the materialized builtin catalog SHALL live under `.homonto/` only. Current adapters resolve local-source skills (`source = "local:<name>"`) from `homonto/skills/<name>`, local-source commands from `homonto/commands/<name>.md`, and local-source subagents from `homonto/subagents/<name>.md`. Builtin-source skills resolve from the materialized `.homonto/catalog/skills/<name>/`, builtin-source commands from `.homonto/catalog/commands/<name>.md`, and builtin-source subagents from `.homonto/catalog/subagents/<name>.md`. Local framework content resolution beyond these resource kinds is part of future framework/catalog projection work and MUST NOT be claimed as installed behavior yet.

#### Scenario: Local skill resolves from homonto/

- **GIVEN** a config with `[skills.my-skill] source = "local:my-skill"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `homonto/skills/my-skill/`

#### Scenario: Builtin skill resolves from materialized catalog

- **GIVEN** a config with `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `.homonto/catalog/skills/brainstorming/`

#### Scenario: Local command resolves from homonto/commands

- **GIVEN** a config with `[commands.mine] source = "local:mine"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `homonto/commands/mine.md`

#### Scenario: Builtin command resolves from materialized catalog

- **GIVEN** a config with `[commands.demo] source = "builtin:demo"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `.homonto/catalog/commands/demo.md`

#### Scenario: Local subagent resolves from homonto/subagents

- **GIVEN** a config with `[subagents.mine] source = "local:mine"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `homonto/subagents/mine.md`

#### Scenario: Builtin subagent resolves from materialized catalog

- **GIVEN** a config with `[subagents.code-reviewer] source = "builtin:code-reviewer"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `.homonto/catalog/subagents/code-reviewer.md`
