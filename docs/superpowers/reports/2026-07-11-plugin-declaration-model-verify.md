# Verification Report: plugin-declaration-model (v1.2 #1)

- **Change**: `plugin-declaration-model` — plugins from bare-name lists → `[plugins.<tool>.<name>]` declaration tables
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (scale: 2 capabilities, 20 changed files)
- **Result**: PASS — one IMPORTANT review finding was fixed during build; no open CRITICAL/IMPORTANT issues

## Scope

Config model + validation (`internal/config/config.go`), both adapters
(`internal/adapter/{claude,opencode}/…`), plugin test migration + new
disable-semantics tests, README + roadmap. `Plugin{Source string; Enabled *bool}`,
`Plugins.{Claude,OpenCode} map[string]Plugin`. Breaking pre-release schema change,
no shim.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md / plan tasks completed `[x]` | PASS (0 unchecked) |
| 2 | Implementation matches `design.md` (Option A source-keyed) | PASS |
| 3 | Implementation matches Design Doc (`docs/superpowers/specs/2026-07-11-plugin-declaration-model-design.md`) | PASS |
| 4 | All delta-spec scenarios pass | PASS (config-model + tool-adapters) |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS (spec patched to source-keyed to match design) |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Capability / scenario | Test | Result |
|---|---|---|
| config: parse declaration tables | `TestLoadPluginEnabledSemantics` | PASS |
| config: no source rejected | `TestLoadRejectsEmptyPluginSource` | PASS |
| config: enabled default true / false disables | `TestLoadPluginEnabledSemantics` | PASS |
| config: reserved plugin keys rejected | existing reserved-key tests + index-like names | PASS |
| config: duplicate source rejected (added post-review) | `TestLoadRejectsDuplicatePluginSource` | PASS |
| claude: disabled → `enabledPlugins[source]=false` | `TestClaudeProjectsPluginEnableDisable` | PASS |
| claude: enabled → `true` | `TestClaudeProjectsPluginEnableDisable` | PASS |
| claude: deterministic plan | `TestClaudePluginPlanIsDeterministic` | PASS |
| opencode: enabled source appended no-dup | `TestOpenCodeEnabledPluginAppendedNoDup` | PASS |
| opencode: disabled managed → removed | `TestOpenCodeDisabledManagedPluginRemoved` | PASS |
| opencode: disabled absent → noop | `TestOpenCodeDisabledPluginAbsentIsNoop` | PASS |
| opencode: unmanaged entries preserved | `TestOpenCodeDisabledUnmanagedEntryPreserved` | PASS |
| opencode: adopt pre-existing | `TestOpenCodeAdoptPluginRecordsState` | PASS |

## Commands run (verification evidence)

| Command | Result |
|---|---|
| `go build ./...` (both binaries) | Success |
| `go test ./... -count=1` | 330 passed, 23 packages |
| `go test -race ./internal/config/... ./internal/adapter/...` | 122 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |
| `go mod tidy` diff | clean |

## E2E (real `homonto` binary, temp $HOME)

`homonto.toml` with a claude plugin enabled + one disabled + an opencode plugin →
`homonto plan`: `plugin.claude-hud@official = true`, `plugin.legacy@official =
false`, opencode source appended. `homonto apply --yes` wrote
`enabledPlugins: {"claude-hud@official": true, "legacy@official": false}` (disable
expressible as a real `false`, not absence) and `opencode.jsonc`
`{"plugin": ["@slkiser/opencode-quota"]}`. A second `plan` reported **"No changes.
Everything up to date."** — idempotent.

## Code review (review_mode: standard) — one IMPORTANT finding, FIXED

The final lightweight review flagged one IMPORTANT issue: two plugin declarations
sharing the same `source` collide on the single source-keyed projection key,
giving a last-writer-wins, map-iteration-order-dependent (non-deterministic)
plan — violating the deterministic-plan invariant. **Fixed** during build: a
per-tool duplicate-source guard now rejects it at load
(`TestLoadRejectsDuplicatePluginSource`). All other checks (OpenCode
disable/no-double-delete via the `declared` set, unmanaged-entry safety, Claude
false-vs-absence idempotency, nil-map safety, test strength) were reviewed and
confirmed correct.

## Conclusion

Verification PASS. First increment of roadmap v1.2 (plugin declaration model +
enable/disable). Deferred follow-ups: per-plugin `config` → Claude
`pluginConfigs`; Claude `extraKnownMarketplaces`; OpenCode `config` handling.
