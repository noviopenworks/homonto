# config-model Specification

## Purpose
Defines the `homonto.toml` desired-state model shared by adapters: MCP servers,
owned skills, per-tool plugins, per-tool settings, target selection, and
unresolved secret references.
## Requirements
### Requirement: Declarative config as single source of truth

`homonto` SHALL parse a single `homonto.toml` file into one tool-agnostic
desired-state model covering MCP servers, owned skills, per-tool plugins, and
per-tool settings. All downstream stages SHALL operate on this model, never on
raw TOML.

#### Scenario: Parse a complete config
- **WHEN** `homonto.toml` declares MCP servers, `[skills] own`, per-tool
  `[plugins]`, and per-tool `[settings]`
- **THEN** the loader returns a model exposing each MCP's command/env/targets,
  the owned skill list, the per-tool plugin lists, and the per-tool settings maps

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

### Requirement: Skill install scope

The config model SHALL expose an optional `[skills] scope` selecting where owned skills
install: `"user"` (the default when absent or empty) targets the user's home tool
directories, and `"project"` targets the project root (the directory of `homonto.toml`).
`config.Load` SHALL reject any other value, naming the offending value and the valid set
(`user`, `project`). Scope SHALL govern skill symlinks only; MCP servers and settings are
unaffected.

#### Scenario: Absent scope defaults to user
- **WHEN** `homonto.toml` has `[skills] own` but no `scope`
- **THEN** the loaded model's skill scope is `user`

#### Scenario: Project scope parsed
- **WHEN** `homonto.toml` sets `[skills] scope = "project"`
- **THEN** the loaded model's skill scope is `project`

#### Scenario: Invalid scope is rejected
- **GIVEN** `[skills] scope = "global"` (not a valid value)
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming `"global"` and the valid values `user` and
  `project`, rather than silently defaulting
