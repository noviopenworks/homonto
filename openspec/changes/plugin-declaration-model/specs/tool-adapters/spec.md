## ADDED Requirements

### Requirement: Plugin enable/disable projection

Both adapters SHALL project declared plugins from the
`[plugins.<tool>.<name>]` model, honoring each plugin's `source` and `enabled`
flag, surgically (unmanaged keys/entries preserved) and idempotently. A plugin's
projected state SHALL be keyed by its declaration name (`plugin.<name>`).

- **Claude**: for each declared Claude plugin, the adapter SHALL set
  `enabledPlugins[<source>]` to the plugin's `enabled` value in
  `settings.json` — writing `false` for a disabled plugin, not only `true`.
  Pruning and adoption of `enabledPlugins.<key>` are unchanged.
- **OpenCode**: an enabled plugin SHALL have its `source` present in the
  `plugin` array (appended without duplication); a disabled plugin
  (`enabled = false`) SHALL be ensured absent from the `plugin` array (pruned if
  present and managed).

Per-plugin `config` projection (Claude `pluginConfigs`) and Claude marketplace
registration (`extraKnownMarketplaces`) are OUT OF SCOPE for this requirement.

#### Scenario: Claude projects a disabled plugin as false

- **GIVEN** a Claude plugin `[plugins.claude.hud]` with `source = "hud@official"` and `enabled = false`
- **WHEN** apply runs
- **THEN** `settings.json` `enabledPlugins["hud@official"]` is `false`, and unrelated keys are preserved

#### Scenario: Claude projects an enabled plugin as true

- **GIVEN** a Claude plugin with `source = "hud@official"` and `enabled` omitted
- **WHEN** apply runs
- **THEN** `enabledPlugins["hud@official"]` is `true`

#### Scenario: OpenCode appends an enabled plugin's source

- **GIVEN** an OpenCode plugin `[plugins.opencode.quota]` with `source = "@slkiser/opencode-quota"` enabled, against an existing `plugin` array
- **WHEN** apply runs
- **THEN** `@slkiser/opencode-quota` is present in the `plugin` array without duplicating existing entries

#### Scenario: OpenCode removes a disabled plugin from the array

- **GIVEN** an OpenCode plugin declared `enabled = false` whose `source` is currently present in the `plugin` array as a homonto-managed entry
- **WHEN** apply runs
- **THEN** the `source` is removed from the `plugin` array
