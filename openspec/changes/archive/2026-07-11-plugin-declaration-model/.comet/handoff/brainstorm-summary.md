# Brainstorm Summary

- Change: plugin-declaration-model
- Date: 2026-07-11

## Confirmed Technical Approach

v1.2 increment 1. Migrate `internal/config` `Plugins{Claude []string; OpenCode
[]string}` → `Plugins{Claude map[string]Plugin; OpenCode map[string]Plugin}`
with `Plugin{ Source string; Enabled *bool }` + `IsEnabled()`. Validation:
`validateKey` on decl name, reject empty source, preserve reserved-key guards.
Claude adapter projects `enabledPlugins[pl.Source] = pl.IsEnabled()` (state key
`plugin.<declName>`, disk key `source`; disable emits `false`). OpenCode adapter
projects enabled→source in `plugin` array, disabled→ensure absent (prune managed
entries). Real tool formats confirmed (see [[plugin-config-formats]]).

## Key Trade-offs and Risks

- State keyed by decl name, disk keyed by source (indirection, covered by tests).
- OpenCode disable = array removal via the shared membership-prune path; must
  only touch homonto-managed entries (state-guarded like other prunes).
- Broad but mechanical test churn (every plugin test moves list→table).
- Breaking pre-release schema change, no shim.

## Testing Strategy

TDD RED first for model+validation and both adapters. Update all existing plugin
tests to table form. Regression (build/test/race/vet/gofmt/mod tidy) + a
plan-idempotency E2E showing enable/disable.

## Spec Patches

None. Delta specs (config-model + tool-adapters ADDED requirements) already
carry the plugin-declaration model, enable/disable, and projection scenarios.
Deferred (follow-up): `config`→pluginConfigs, extraKnownMarketplaces, OpenCode
config handling.
