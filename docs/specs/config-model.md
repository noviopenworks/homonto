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
supported tools; an MCP with an explicit `targets` list SHALL apply only to those
tools. Unsupported target names currently match no adapter and are silently
ignored; validation for typos such as `claud` is a known gap.

#### Scenario: No targets means all tools
- **WHEN** an MCP entry omits `targets`
- **THEN** its effective targets are `["claude", "opencode"]`

#### Scenario: Explicit targets are honored
- **WHEN** an MCP entry sets `targets = ["claude"]`
- **THEN** its effective targets are exactly `["claude"]`

#### Scenario: Unsupported target is ignored
- **WHEN** an MCP entry sets `targets = ["claud"]`
- **THEN** no current adapter projects that MCP, because only `claude` and
  `opencode` are recognized by adapter matching

### Requirement: Secret references preserved as unresolved tokens

The config model SHALL retain secret references (`${pass:…}`, `${ENV}`) verbatim
as unresolved tokens; parsing SHALL NOT resolve them.

#### Scenario: Env value with a pass reference
- **WHEN** an MCP `env` value is `"${pass:ai/brave}"`
- **THEN** the parsed model stores `"${pass:ai/brave}"` unchanged
