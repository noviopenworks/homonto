## Context

v1.2 #1 landed the `[plugins.<tool>.<name>]` model with `source`/`enabled`. This
increment (#2) adds the `config` half. Claude stores per-plugin non-sensitive
settings at `pluginConfigs.<plugin>.options` (see the `plugin-config-formats`
research). OpenCode has no per-plugin config on disk, so OpenCode `config` is
rejected at load. Claude marketplace registration (`extraKnownMarketplaces`) is a
separate follow-up (needs a marketplace-declaration model).

## Goals / Non-Goals

**Goals**: `Plugin.Config map[string]any`; project Claude `config` →
`pluginConfigs.<source>.options` (new `pluginconfig.` managed namespace: desired,
read-back, apply, prune, adopt, deterministic); reject OpenCode `config` at load.

**Non-Goals**: Claude `extraKnownMarketplaces` (follow-up); OpenCode config
projection (no native home); secret-bearing plugin config values (config is
non-sensitive, same as `settings`).

## Decisions

### D1 — New Claude managed namespace `pluginconfig.<source>`

Mirror the existing `plugin.`/`setting.` key-namespace pattern in
`internal/adapter/claude/claude.go`:
- **desired** (`desired()`): `for _, pl := range c.Plugins.Claude { if len(pl.Config) > 0 { out["pluginconfig."+pl.Source] = mustJSON(map[string]any{"options": pl.Config}) } }`.
- **read-back** (`current()`): `for k, v := range objMembers(sj, "pluginConfigs") { out["pluginconfig."+k] = v }`, and add `"pluginConfigs"` to the skip set in the generic settings read-back loop.
- **apply write**: `case hasPrefix(c.Key, "pluginconfig."): sj = SetJSON(sj, "pluginConfigs."+EscapePath(trim(c.Key,"pluginconfig.")), val)`.
- **prune**: `case hasPrefix(c.Key, "pluginconfig."): sj = DeleteJSON(sj, "pluginConfigs."+EscapePath(trim(c.Key,"pluginconfig.")))`.
- **managed prefix** (`util.go`): add `"pluginconfig."` to the recognized prefix list.
- adoption is handled by the generic adopt path (records desired → hash of
  on-disk value) with no special case.

The value stored/compared is the whole `{"options": {...}}` object, so read-back
(`objMembers` yields each `pluginConfigs.<k>` member verbatim) and desired agree,
giving idempotent plans.

### D2 — OpenCode config rejected at load

In `config.go` validation, for each `opencode` plugin with `len(pl.Config) > 0`
return an error: OpenCode plugins are a plain array with no per-plugin config; do
not silently drop it. Claude plugins may carry `config`.

### D3 — Reserved key

Add `pluginConfigs` to the `settings.claude` reserved-key rejection (alongside
`enabledPlugins`, `mcpServers`).

## Risks / Trade-offs

- **Ordering with `enabledPlugins` and settings in the same file**: three managed
  namespaces (`enabledPlugins`, `pluginConfigs`, top-level settings) now coexist
  in `settings.json`; read-back must exclude BOTH `enabledPlugins` and
  `pluginConfigs` from the generic settings loop (or they'd double-count as
  `setting.` keys, breaking idempotency). Covered by a mixed test.
- **`config` value canonicalization**: `map[string]any` → `mustJSON` →
  `jsonutil.Canonical` for the state hash, same path as `setting` values, so
  key-order noise doesn't churn plans.

## Migration Plan

Additive within v1.2; `config` is optional. No migration.

## Open Questions

None. Marketplace registration is a deferred, scoped follow-up.
