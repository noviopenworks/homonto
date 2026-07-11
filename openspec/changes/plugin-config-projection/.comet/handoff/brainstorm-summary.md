# Brainstorm Summary

- Change: plugin-config-projection
- Date: 2026-07-11

## Confirmed Technical Approach

v1.2 #2. Add `Config map[string]any` to `Plugin`. New Claude managed namespace
`pluginconfig.<source>` → `pluginConfigs.<source>.options` in settings.json
(desired/read-back/apply/prune/managed-prefix, mirroring the `plugin.`/`setting.`
pattern). Exclude `pluginConfigs` from the generic settings read-back (alongside
enabledPlugins/mcpServers). Reject OpenCode `config` at load (no native home).
Add `pluginConfigs` to settings.claude reserved keys. Formats: see
[[plugin-config-formats]].

## Key Trade-offs and Risks

- Three managed namespaces now coexist in settings.json (enabledPlugins,
  pluginConfigs, top-level settings); read-back MUST exclude both enabledPlugins
  AND pluginConfigs from the generic settings loop or idempotency breaks.
- config canonicalized via mustJSON→Canonical for the state hash (same as
  settings) so key-order noise doesn't churn plans.

## Testing Strategy

TDD RED first (config model+validation; claude projection). E2E: claude plugin
with config → pluginConfigs on disk, idempotent re-plan; opencode config rejected.
Full regression.

## Spec Patches

None. Delta specs (config-model MODIFIED + tool-adapters ADDED) already carry the
config field, opencode rejection, and pluginConfigs projection scenarios.
Deferred: Claude extraKnownMarketplaces (marketplace-declaration model).
