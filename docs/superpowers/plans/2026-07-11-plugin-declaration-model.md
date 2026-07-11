---
change: plugin-declaration-model
design-doc: docs/superpowers/specs/2026-07-11-plugin-declaration-model-design.md
base-ref: 4d2da77c910b1714404aabe2f0dcbcaaf71c4640
---

# Plan: plugin declaration model (v1.2 #1)

Migrate plugins from bare-name lists to `[plugins.<tool>.<name>]` declaration
tables (`source` + `enabled`); teach both adapters enable/disable. See the
Design Doc for the exact Go changes (Option A: source-keyed state, decl name is
an organizational label this increment) and the deferred follow-ups. TDD.

## Task 1: config model + validation (`internal/config`)

- [ ] 1.1 (TDD RED first) `type Plugin { Source string \`toml:"source"\`; Enabled *bool \`toml:"enabled"\` }` + `(Plugin) IsEnabled() bool` (`Enabled==nil || *Enabled`); `Plugins{ Claude map[string]Plugin \`toml:"claude"\`; OpenCode map[string]Plugin \`toml:"opencode"\` }`.
- [ ] 1.2 (TDD RED first) Validation in Parse/Load: range both tools' plugin maps, `validateKey("plugins.<tool>", declName)` + reject empty `Source` (error naming the plugin); preserve `settings.claude.enabledPlugins` / `settings.opencode.plugin`/`mcp` reserved-key guards. Tests: parse table form (source+enabled, enabled-omitted→true); empty source rejected; `enabled=false`→disabled; reserved keys still rejected.
- [ ] 1.3 GREEN; gofmt/vet clean for `internal/config`. Commit: `feat(config): plugin declaration tables ([plugins.<tool>.<name>] source+enabled)`

## Task 2: Claude adapter (`internal/adapter/claude`)

- [ ] 2.1 (TDD RED first) Replace `for _, p := range c.Plugins.Claude { out["plugin."+p]=\`true\` }` with `for _, pl := range c.Plugins.Claude { out["plugin."+pl.Source] = mustJSON(pl.IsEnabled()) }` (source-keyed; disabled emits `false`). Read-back/prune of `enabledPlugins.<key>` already source-keyed — leave unchanged. Tests: enabled→`enabledPlugins[source]=true`; disabled→`=false`; unrelated keys preserved; consecutive plans byte-identical (deterministic).
- [ ] 2.2 GREEN; gofmt/vet clean. Commit: `feat(claude): project plugin enable/disable from declaration model`

## Task 3: OpenCode adapter (`internal/adapter/opencode`)

- [ ] 3.1 (TDD RED first) Range `c.Plugins.OpenCode` map: enabled → adopt/create `plugin.<source>` with array value `pl.Source` (as today, source-keyed); disabled (`!IsEnabled()`) → if `arrayHas(doc,"plugin",pl.Source)` AND recorded in state, emit a delete/prune removing it; else noop. Mirror in the apply path (~line 412). Tests: enabled→source appended no-dup; disabled managed→removed; disabled-absent→noop; unmanaged entries preserved; adopt pre-existing.
- [ ] 3.2 GREEN; gofmt/vet clean. Commit: `feat(opencode): project plugin enable/disable from declaration model`

## Task 4: Test migration, regression, docs

- [ ] 4.1 Update every remaining plugin test (`internal/config/config_test.go`, `internal/adapter/claude/*_test.go`, `internal/adapter/opencode/*_test.go`) from list form to table form (`map[string]config.Plugin{"n":{Source:"n"}}`).
- [ ] 4.2 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E: a `homonto.toml` with a claude plugin (one enabled + one disabled) and an opencode plugin → `homonto plan` shows correct enable/disable; a second `plan` is byte-identical.
- [ ] 4.3 Update `docs/roadmap.md` (v1.2 plugin declaration model landed — first increment; config/marketplace projection next) + any README/config docs showing the old `[plugins] claude=[...]` list form. No over-claim.
- [ ] 4.4 Commit all changes.
