# Delta Spec: config-model (validate-config)

## MODIFIED Requirements

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

## ADDED Requirements

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
