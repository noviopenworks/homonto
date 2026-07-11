# Verification Report: plugin-config-projection (v1.2 #2)

- **Change**: `plugin-config-projection` — per-plugin `config` → Claude `pluginConfigs.<source>.options`
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (2 capabilities, config + adapter changes)
- **Result**: PASS — no CRITICAL/IMPORTANT issues (final review: no findings)

## Scope

`internal/config/config.go` (`Plugin.Config`, OpenCode-config rejection,
`settings.claude.pluginConfigs` reserved), `internal/adapter/claude/{claude,util}.go`
(new `pluginconfig.<source>` managed namespace → `pluginConfigs.<source>.options`),
tests, README + roadmap.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` decisions (D1 namespace, D2 OpenCode reject, D3 reserved key) | PASS |
| 3 | Matches Design Doc (exact desired/read-back/apply/prune edits + exclusion hazard) | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| Claude plugin config parsed | `TestLoadPluginConfig` | PASS |
| OpenCode plugin config rejected | `TestLoadRejectsOpenCodePluginConfig` | PASS |
| `settings.claude.pluginConfigs` reserved | config reserved-key test | PASS |
| config → `pluginConfigs[source].options` | `TestClaudeProjectsPluginConfig` | PASS |
| no config → no pluginConfigs entry | `TestClaudeProjectsPluginConfig` (no-config case) | PASS |
| de-declared config pruned | `TestClaudePluginConfigDeDeclared` | PASS |
| adopt pre-existing pluginConfigs | `TestClaudeAdoptsPluginConfig` | PASS |
| deterministic plan | `TestClaudePluginConfigPlanDeterministic` | PASS |
| enabled+config (no setting.* leak) | `TestClaudePluginConfigWithEnabled` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 337 passed, 23 packages |
| `go test -race ./internal/config/... ./internal/adapter/claude/...` | 86 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, temp $HOME)

A `[plugins.claude.hud]` with `enabled=true` + `config={api_endpoint,max_workers}`
→ `plan` shows `plugin.hud@official = true` and `pluginconfig.hud@official =
{"options":{…}}`; `apply` wrote `enabledPlugins:{"hud@official":true}` AND
`pluginConfigs:{"hud@official":{"options":{"api_endpoint":"https://x.example","max_workers":4}}}`;
a second `plan` reported **"No changes. Everything up to date."** (idempotent —
the read-back exclusion hazard is handled). An OpenCode plugin declaring `config`
failed `plan` with the rejection message.

## Code review (review_mode: standard) — no findings

The final review verified all six risk areas correct: the `pluginConfigs`
read-back exclusion (idempotency), the `pluginconfig.`-vs-`plugin.` prefix
non-collision, prune behavior (config-only de-declare keeps `enabledPlugins`;
full de-declare prunes both), `@`-in-source escaping symmetry across
desired/apply/prune/read-back, the OpenCode-only rejection gating (empty config
passes), and the reserved-key guard. No CRITICAL/IMPORTANT/MINOR fixes required.

## Conclusion

Verification PASS. v1.2 #2 complete (per-plugin config projection). Remaining v1.2
increment: Claude marketplace registration (`extraKnownMarketplaces`), which needs
a marketplace-declaration model.
