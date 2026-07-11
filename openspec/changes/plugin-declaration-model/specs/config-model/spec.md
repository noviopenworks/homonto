## ADDED Requirements

### Requirement: Plugin declaration model

Plugins SHALL be declared as per-tool, per-plugin tables
`[plugins.<tool>.<name>]` (tool ∈ {`claude`, `opencode`}), replacing the prior
bare-name lists (`[plugins] claude = [...]`). This is a breaking, pre-release
schema change with no migration shim. Each plugin table SHALL carry:

- `source` (required, non-empty string): the tool-native plugin identifier —
  for `claude` the `name@marketplace` key used in `enabledPlugins`; for
  `opencode` the npm package name or local plugin path placed in the `plugin`
  array.
- `enabled` (optional boolean, default `true`): `false` marks the plugin
  disabled.

The declaration name (the table key) and the `source` SHALL be validated with
the same key-validation guard applied to other config keys. A plugin whose
`source` is empty SHALL be rejected. The existing reserved-key guards SHALL be
preserved: `settings.claude.enabledPlugins` and `settings.opencode.plugin` (and
`mcp`) remain rejected because homonto manages those structures.

#### Scenario: Parse plugin declaration tables

- **GIVEN** a config with `[plugins.claude.claude-hud]` (`source = "claude-hud@official"`, `enabled = true`) and `[plugins.opencode.quota]` (`source = "@slkiser/opencode-quota"`, no `enabled`)
- **WHEN** the config is parsed
- **THEN** it yields a Claude plugin `claude-hud` with source `claude-hud@official` enabled, and an OpenCode plugin `quota` with source `@slkiser/opencode-quota` whose enabled defaults to true

#### Scenario: A plugin without a source is rejected

- **GIVEN** a `[plugins.claude.x]` table with no `source` (or an empty `source`)
- **WHEN** the config is parsed
- **THEN** it is rejected with an error identifying the plugin

#### Scenario: enabled defaults to true and false disables

- **GIVEN** one plugin with `enabled = false` and one with `enabled` omitted
- **WHEN** the config is parsed
- **THEN** the first is disabled and the second is enabled (default true)

#### Scenario: Reserved plugin settings keys still rejected

- **GIVEN** a `settings.claude` key `enabledPlugins` or a `settings.opencode` key `plugin`
- **WHEN** the config is parsed
- **THEN** it is rejected as reserved (homonto manages plugins there)
