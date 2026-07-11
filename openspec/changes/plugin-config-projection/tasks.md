## 1. Config model + validation (`internal/config`)

- [x] 1.1 (TDD RED first) Add `Config map[string]any \`toml:"config"\`` to `Plugin`.
- [x] 1.2 (TDD RED first) Validation: reject an `opencode` plugin with a non-empty `config` (error naming the plugin, explaining OpenCode has no per-plugin config); add `pluginConfigs` to the `settings.claude` reserved-key rejection. Tests: claude plugin with config parses; opencode plugin with config rejected; `settings.claude.pluginConfigs` rejected; existing enable/disable + dup-source guards still pass.
- [x] 1.3 GREEN; gofmt/vet clean for `internal/config`. Commit: `feat(config): plugin config field + opencode-config/pluginConfigs guards`

## 2. Claude pluginConfigs projection (`internal/adapter/claude`)

- [x] 2.1 (TDD RED first) Add the `pluginconfig.` managed namespace per the Design Doc: desired (`out["pluginconfig."+pl.Source] = mustJSON({"options": pl.Config})` for each claude plugin with non-empty config); read-back (`objMembers(sj,"pluginConfigs")` → `pluginconfig.<k>`, and exclude `pluginConfigs` from the generic settings read-back loop alongside `enabledPlugins`/`mcpServers`); apply write (`SetJSON pluginConfigs.<EscapePath(source)>`); prune (`DeleteJSON pluginConfigs.<...>`); add `"pluginconfig."` to `util.go` managed prefixes.
- [x] 2.2 (TDD RED first) Tests: config projected → `pluginConfigs[source].options.<k>` on disk after apply; no config → no `pluginConfigs` entry; de-declared config pruned; adopt a pre-existing matching `pluginConfigs.<source>`; unrelated settings + other pluginConfigs entries preserved; consecutive plans byte-identical; a plugin with BOTH enabled+config projects `enabledPlugins[source]` AND `pluginConfigs[source].options` without either read-back double-counting as a `setting.`.
- [x] 2.3 GREEN; gofmt/vet clean. Commit: `feat(claude): project per-plugin config to pluginConfigs.<source>.options`

## 3. Regression and docs

- [x] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E: a `homonto.toml` with a claude plugin carrying `config` → `homonto plan`/`apply` writes `pluginConfigs.<source>.options`; a second `plan` is byte-identical; an opencode plugin with `config` fails `homonto plan` with the rejection message.
- [x] 3.2 Update `docs/roadmap.md` v1.2 status (per-plugin config projection landed; marketplace registration is the remaining v1.2 increment) + README `[plugins.claude.<name>]` example to show an optional `config`. No over-claim.
- [x] 3.3 Commit all changes.
