## Why

Roadmap v1.2 (Plugin Configuration) expands plugin support "from simple
references to declarations with configuration." Today `homonto.toml` declares
plugins as bare name lists — `[plugins] claude = ["x"]`, `opencode = ["y"]` —
and homonto can only *enable* a Claude plugin (`enabledPlugins.<name> = true`)
or *add* an OpenCode plugin (a `plugin` array entry). There is no way to
disable a plugin, or to bind a Claude plugin to its marketplace, or to carry
per-plugin config. This change lays the foundation: it replaces the bare-list
model with per-plugin declaration tables `[plugins.<tool>.<name>]` carrying a
`source` and an `enabled` flag, and teaches both adapters enable/disable
semantics. It is the first increment of v1.2; per-plugin `config` projection
(Claude `pluginConfigs`) and Claude marketplace registration
(`extraKnownMarketplaces`) are explicit follow-ups.

## What Changes

- **BREAKING (pre-release)**: the plugin config model changes from string lists
  to declaration tables. `internal/config`'s `Plugins{Claude []string;
  OpenCode []string}` becomes `Plugins{Claude map[string]Plugin; OpenCode
  map[string]Plugin}` with `type Plugin { Source string; Enabled *bool }`. In
  TOML:
  ```toml
  # before
  [plugins]
  claude = ["claude-hud"]
  opencode = ["opencode-quota"]
  # after
  [plugins.claude.claude-hud]
  source = "claude-hud@official"
  enabled = true
  [plugins.opencode.opencode-quota]
  source = "@slkiser/opencode-quota"
  # enabled defaults to true
  ```
  The map key is the plugin's declaration name; `source` is the tool-native
  identifier (Claude: the `name@marketplace` key used in `enabledPlugins`;
  OpenCode: the npm package / local plugin path placed in the `plugin` array).
  `enabled` is optional and defaults to `true`; `false` disables the plugin.
- **Validation**: each plugin's `source` is required (non-empty); the
  declaration name and source are validated as keys (same `validateKey` guard as
  today). The existing `settings.claude.enabledPlugins` and
  `settings.opencode.plugin`/`mcp` reserved-key guards are preserved.
- **Claude adapter**: projects `enabledPlugins[<source>] = <enabled>` for each
  declared Claude plugin — now writing `false` for a disabled plugin, not only
  `true`. Prune/adopt of `enabledPlugins.<key>` is unchanged.
- **OpenCode adapter**: an enabled plugin places its `source` in the `plugin`
  array (as today); a disabled plugin (`enabled = false`) is ensured absent
  (pruned if present). State is keyed by the declaration name.
- Existing plugin tests across `internal/config` and both adapters are updated
  to the new model.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `plugin-configuration`: plugins are declared as `[plugins.<tool>.<name>]`
  tables with a required `source` and an optional `enabled` flag (default true);
  homonto projects enable **and** disable to each tool's native plugin config.

## Impact

- `internal/config/config.go`: `Plugin` type; `Plugins` maps; validation.
- `internal/adapter/claude/claude.go`: `enabledPlugins` projection honors
  `source` + `enabled` (incl. `false`).
- `internal/adapter/opencode/opencode.go`: `plugin` array projection honors
  `source` + `enabled` (disabled → absent).
- Tests updated in `internal/config` and both adapters.
- **BREAKING** `homonto.toml` schema (pre-release, no migration shim); documented
  in the change. No new dependency.
- Follow-ups (NOT in scope): per-plugin `config` → Claude `pluginConfigs`;
  Claude `extraKnownMarketplaces` registration; OpenCode `config` handling.
