## 1. Config model + validation (`internal/config`)

- [ ] 1.1 (TDD, RED first) Add `type Plugin struct { Source string \`toml:"source"\`; Enabled *bool \`toml:"enabled"\` }` and an `(Plugin) IsEnabled() bool` helper (`Enabled == nil || *Enabled`). Change `Plugins` to `{ Claude map[string]Plugin \`toml:"claude"\`; OpenCode map[string]Plugin \`toml:"opencode"\` }`.
- [ ] 1.2 (TDD, RED first) Validation in `Parse`/`Load`: for each tool's plugins, `validateKey("plugins.<tool>", declName)` and reject empty `source` (error naming the plugin). Preserve the `settings.claude.enabledPlugins` and `settings.opencode.plugin`/`mcp` reserved-key guards. Tests: parse a claude+opencode plugin table (source+enabled, and enabled-omitted→true); empty source rejected; `enabled=false` parses as disabled; reserved settings keys still rejected.
- [ ] 1.3 GREEN; gofmt/vet clean for `internal/config`.

## 2. Adapter projection (`internal/adapter/{claude,opencode}`)

- [ ] 2.1 (TDD, RED first) Claude (`claude.go`): in the desired-values builder replace `out["plugin."+p]=\`true\`` — for each `name, pl := range c.Plugins.Claude`, set `out["plugin."+name] = mustJSON(pl.IsEnabled())`, and project that value at `enabledPlugins[pl.Source]` on apply (the `plugin.` key path already maps to `enabledPlugins.<...>`; make the on-disk key `pl.Source`, not the decl name). Prune/adopt of `enabledPlugins.<key>` unchanged. Tests: enabled plugin → `enabledPlugins[source]=true`; disabled → `enabledPlugins[source]=false`; unrelated keys preserved; idempotent re-plan.
- [ ] 2.2 (TDD, RED first) OpenCode (`opencode.go`): for each `name, pl := range c.Plugins.OpenCode` — if enabled, adopt/create with array value `pl.Source` and state key `plugin.<name>` (as today); if disabled, ensure `pl.Source` absent (prune/delete when present & managed, else noop). Tests: enabled → source appended without dup; disabled managed entry → removed; disabled-not-present → noop; existing unmanaged array entries preserved.
- [ ] 2.3 GREEN; gofmt/vet clean for both adapters.

## 3. Test migration, regression, docs

- [ ] 3.1 Update all existing plugin tests (`internal/config/config_test.go`, `internal/adapter/claude/*_test.go`, `internal/adapter/opencode/*_test.go`) from the list form to the new table form so they compile and assert the new behavior.
- [ ] 3.2 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E: a `homonto.toml` with a claude plugin (enabled + a disabled one) and an opencode plugin → `homonto plan` shows the plugin changes with correct enable/disable; a second `plan` is byte-identical (idempotent).
- [ ] 3.3 Update `docs/roadmap.md` (v1.2: plugin declaration model landed — first increment; `config`/marketplace projection are the next increments) and any README/config docs showing the old `[plugins] claude = [...]` list form. No over-claim.
- [ ] 3.4 Commit all changes.
