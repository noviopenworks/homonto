# config-model Specification

## Purpose
Defines the `homonto.toml` desired-state model shared by adapters: MCP servers,
explicit framework/skill/command/subagent resources, per-tool plugins, per-tool
settings, target selection, model routing, and unresolved secret references.
## Requirements
### Requirement: Declarative config as single source of truth

`homonto` SHALL parse a single `homonto.toml` file into one tool-agnostic
desired-state model covering MCP servers, explicit framework/skill/command/
subagent resources, per-tool plugins, per-tool settings, and model routing. All
downstream stages SHALL operate on this model, never on raw TOML.

#### Scenario: Parse a complete config
- **WHEN** `homonto.toml` declares MCP servers, explicit resource tables
  (`[frameworks.<name>]`, `[skills.<name>]`, `[commands.<name>]`,
  `[subagents.<name>]`), per-tool `[plugins]`, per-tool `[settings]`, and needed
  `[models.<tool>.<level>]` routes
- **THEN** the loader returns a model exposing each MCP's command/env/targets,
  the resources (each with source, scope, and targets), the per-tool plugin
  lists, the per-tool settings maps, and model routes

#### Scenario: Missing config file is an error
- **WHEN** the config path does not exist
- **THEN** `Load` returns an error rather than an empty model

### Requirement: MCP target defaulting

An MCP server declared without an explicit `targets` list SHALL apply to all
supported tools; an MCP with an explicit `targets` list SHALL apply only to
those tools. Every listed target MUST name a supported tool (`claude` or
`opencode`); `config.Load` SHALL reject an unknown target name, naming the
offending value and the valid set.

#### Scenario: No targets means all tools
- **WHEN** an MCP entry omits `targets`
- **THEN** its effective targets are `["claude", "opencode"]`

#### Scenario: Explicit targets are honored
- **WHEN** an MCP entry sets `targets = ["claude"]`
- **THEN** its effective targets are exactly `["claude"]`

#### Scenario: Unknown target is rejected
- **GIVEN** an MCP entry with `targets = ["claud"]` (a typo)
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming `"claud"` and the valid targets
  `claude` and `opencode`, rather than silently projecting the MCP nowhere

### Requirement: Secret references preserved as unresolved tokens

The config model SHALL retain secret references (`${pass:…}`, `${ENV}`) verbatim
as unresolved tokens; parsing SHALL NOT resolve them.

#### Scenario: Env value with a pass reference
- **WHEN** an MCP `env` value is `"${pass:ai/brave}"`
- **THEN** the parsed model stores `"${pass:ai/brave}"` unchanged

### Requirement: Config input validation

`config.Load` SHALL reject a declared MCP that has no command, and SHALL reject
a per-tool settings key that collides with a structure homonto manages in that
tool's file — naming the offending entry in each case — so that unprojectable
or colliding config fails fast at load rather than being silently ignored at
apply.

#### Scenario: MCP without a command is rejected
- **GIVEN** an MCP entry with no `command` (or `command = []`)
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming that MCP, because an MCP with no
  command cannot be projected to any tool

#### Scenario: Reserved settings key is rejected
- **GIVEN** a `settings.claude` key `enabledPlugins`, or a `settings.opencode`
  key `mcp` or `plugin`
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming the reserved key, because homonto
  manages that structure in the same tool file

#### Scenario: Non-colliding settings keys still load
- **GIVEN** settings keys that do not collide (e.g. `settings.claude.model`, or
  `settings.opencode.enabledPlugins`, which is reserved only for claude)
- **WHEN** the config is loaded
- **THEN** `Load` accepts them

### Requirement: Explicit resource declarations

`homonto` SHALL parse frameworks, skills, commands, and subagents as explicit
per-resource tables. Every resource SHALL declare `source` and `scope`. Scope
SHALL be either `user` or `project`; there is no default. Source SHALL be either
`builtin:<name>` or `local:<name>` in the first release.

#### Scenario: Parse explicit resources
- **WHEN** `homonto.toml` declares `[skills.graphify]` with `source = "local:graphify"` and `scope = "project"`
- **THEN** the loader returns a skill resource named `graphify` with local source `graphify` and project scope

#### Scenario: Missing scope is rejected
- **WHEN** a resource omits `scope`
- **THEN** `Load` returns an error naming that resource and the missing scope

### Requirement: Tool-specific model routing

For every model-enabled target tool, `homonto.toml` SHALL define all three model
levels: `architectural`, `coding`, and `trivial`. Each route SHALL include a
non-empty `model` and at least one of `effort` or `variant`. Homonto SHALL not
validate provider-specific model names or effort values beyond presence.

#### Scenario: Model routing for one tool
- **GIVEN** a config whose only model-enabled tool is `claude`
- **WHEN** the loader parses `[models.claude.architectural]`, `[models.claude.coding]`, and `[models.claude.trivial]`, each with a `model` plus `effort` or `variant`
- **THEN** the loader accepts the config and exposes the three routes keyed by tool and level

#### Scenario: Missing model level is rejected
- **WHEN** a model-enabled tool lacks one of the three levels, or a level omits `model`, or a level has neither `effort` nor `variant`
- **THEN** `Load` returns an error naming the offending tool and level

### Requirement: Local provider content root

Local provider content SHALL live under `homonto/` relative to the directory
containing `homonto.toml`; generated state and cache SHALL live under
`.homonto/` only. Current adapters resolve local-source skills
(`source = "local:<name>"`) from `homonto/skills/<name>`. Local command,
subagent, and framework content resolution is part of the future
framework/catalog projection work and MUST NOT be claimed as installed behavior
yet.
