## Why

The plugin declaration model (v1.2 #1) landed `[plugins.<tool>.<name>]` tables
with `source` + `enabled`, but not the `config` half of "declarations with
configuration." Claude Code stores non-sensitive per-plugin settings on disk as
`pluginConfigs.<plugin>.options`; homonto does not project there yet. This
change (v1.2 #2) adds a per-plugin `config` field and projects it to Claude
`pluginConfigs`. OpenCode has no native per-plugin config location (its plugins
are a plain `plugin` string array), so a `config` on an OpenCode plugin is
rejected at load rather than silently dropped — honoring the roadmap non-goal of
"no cross-tool abstraction that hides real Claude/OpenCode plugin differences."

## What Changes

- Add `Config map[string]any` (`toml:"config"`) to `internal/config`'s `Plugin`.
- **Validation**: an OpenCode plugin declaring a non-empty `config` is rejected
  at load with a clear message (OpenCode has no per-plugin config on disk).
  Claude plugins may declare `config`.
- **Claude adapter** gains a new managed key namespace `pluginconfig.<source>`
  that projects to `pluginConfigs.<source>.options` in `settings.json`:
  - desired: each Claude plugin with a non-empty `config` contributes
    `pluginconfig.<source> = {"options": <config>}`;
  - read-back: `pluginConfigs` members are read back as `pluginconfig.<key>`
    (and `pluginConfigs` is excluded from the generic settings read-back);
  - apply writes the `{options: …}` object at `pluginConfigs.<source>`;
  - prune deletes `pluginConfigs.<source>` when the config is de-declared;
  - adoption of a pre-existing `pluginConfigs.<source>` works like other keys;
  - `pluginconfig.` is added to the managed-prefix set.
- Surgical + idempotent: unrelated `settings.json` keys and other
  `pluginConfigs` entries are preserved; consecutive plans are byte-identical.
- `settings.claude.pluginConfigs` is added to the reserved-settings-keys guard
  (homonto now manages that structure).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-model`: the `Plugin` table gains an optional `config` map;
  OpenCode plugins with `config` are rejected.
- `tool-adapters`: the Claude adapter projects per-plugin `config` to
  `pluginConfigs.<source>.options` (surgical, idempotent, pruned, adoptable).

## Impact

- `internal/config/config.go`: `Plugin.Config`; OpenCode-config rejection;
  `settings.claude.pluginConfigs` reserved-key guard.
- `internal/adapter/claude/claude.go`: `pluginconfig.` namespace across
  desired/current/apply/prune; `internal/adapter/claude/util.go`: managed prefix.
- Tests in `internal/config` and `internal/adapter/claude`.
- No new dependency. No OpenCode adapter change (config rejected upstream).
- Deferred to a later increment: Claude marketplace registration
  (`extraKnownMarketplaces`), which needs a marketplace-declaration model.
