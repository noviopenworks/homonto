---
change: plugin-config-projection
design-doc: docs/superpowers/specs/2026-07-11-plugin-config-projection-design.md
base-ref: 1656a0e3338f9016fa4a914f32aeb14c5809e483
archived-with: 2026-07-11-plugin-config-projection
---

# Plan: plugin config projection (v1.2 #2)

Add per-plugin `config`; project Claude `config` → `pluginConfigs.<source>.options`
via a new `pluginconfig.` managed namespace; reject OpenCode `config` at load.
See the Design Doc for exact Go edits and the read-back exclusion hazard. TDD.

## Task 1: config model + validation (`internal/config`)

- [x] 1.1 (TDD RED first) Add `Config map[string]any \`toml:"config"\`` to `Plugin`.
- [x] 1.2 (TDD RED first) Reject an `opencode` plugin with non-empty `config` (error naming the plugin + why); add `pluginConfigs` to the `settings.claude` reserved-key rejection. Tests: claude plugin with config parses; opencode config rejected; `settings.claude.pluginConfigs` rejected; existing enable/disable + dup-source guards still pass.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): plugin config field + opencode-config/pluginConfigs guards`

## Task 2: Claude pluginConfigs projection (`internal/adapter/claude`)

- [x] 2.1 (TDD RED first) Add the `pluginconfig.` namespace (Design Doc D1): desired (`out["pluginconfig."+pl.Source]=mustJSON({"options":pl.Config})` for claude plugins with non-empty config); read-back (`objMembers(sj,"pluginConfigs")`→`pluginconfig.<k>` AND exclude `pluginConfigs` from the generic settings loop next to enabledPlugins/mcpServers); apply write (`SetJSON pluginConfigs.<EscapePath(source)>`); prune (`DeleteJSON pluginConfigs.<...>`); `util.go` managed prefix `"pluginconfig."`.
- [x] 2.2 (TDD RED first) Tests: config→`pluginConfigs[source].options.<k>` on disk after apply; no config→no entry; de-declared config pruned; adopt pre-existing matching entry; unrelated settings + other pluginConfigs preserved; consecutive plans byte-identical; a plugin with enabled+config projects BOTH `enabledPlugins[source]` AND `pluginConfigs[source].options` with NO `setting.pluginConfigs`/`setting.enabledPlugins` leaking (idempotency hazard).
- [x] 2.3 GREEN; gofmt/vet clean. Commit: `feat(claude): project per-plugin config to pluginConfigs.<source>.options`

## Task 3: Regression and docs

- [x] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): a claude plugin with `config` → `apply` writes `pluginConfigs.<source>.options`, second `plan` byte-identical; an opencode plugin with `config` fails `plan` with the rejection message.
- [x] 3.2 Update `docs/roadmap.md` v1.2 status (per-plugin config projection landed; marketplace registration is the remaining v1.2 increment) + README `[plugins.claude.<name>]` example to show an optional `config`. No over-claim.
- [x] 3.3 Commit all changes.
