## 1. Config model + validation (`internal/config`)

- [x] 1.1 (TDD RED first) Add `Marketplace{Source,Repo,URL,Path string; AutoUpdate *bool}` (toml tags incl `auto_update`), `Marketplaces{Claude map[string]Marketplace}`, and `Config.Marketplaces` (`toml:"marketplaces"`).
- [x] 1.2 (TDD RED first) Validation: per claude marketplace, `validateKey("marketplaces.claude", name)`; `source` ∈ {github,url,git-subdir,directory}; required locator present (github→repo, url→url, git-subdir→url+path, directory→path); unknown source or missing locator → error naming the marketplace. Add `extraKnownMarketplaces` to the `settings.claude` reserved-key rejection. Tests: parse github marketplace; unknown source rejected; missing repo (github) rejected; missing url/path (git-subdir) rejected; `settings.claude.extraKnownMarketplaces` rejected.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [marketplaces.claude.<name>] declaration model + validation`

## 2. Claude marketplace projection (`internal/adapter/claude`)

- [x] 2.1 (TDD RED first) Add the `marketplace.` namespace (Design Doc D3): desired (`out["marketplace."+name] = mustJSON(marketplaceValue(mk))`, source sub-object with only type-relevant locator fields + optional autoUpdate); read-back (`objMembers(sj,"extraKnownMarketplaces")`→`marketplace.<k>` AND exclude `extraKnownMarketplaces` from the generic settings loop, now skipping mcpServers/enabledPlugins/pluginConfigs/extraKnownMarketplaces); apply write (`SetJSON extraKnownMarketplaces.<EscapePath(name)>`); prune (`DeleteJSON …`); `util.go` managed prefix `"marketplace."`.
- [x] 2.2 (TDD RED first) Tests: github marketplace → `extraKnownMarketplaces[name].source == {"source":"github","repo":…}` on disk; de-declared marketplace pruned; adopt pre-existing matching entry; unrelated settings + other marketplaces preserved; consecutive plans byte-identical; a settings.json holding a setting + a plugin + a pluginConfig + a marketplace re-plans byte-identical (no namespace leaks); autoUpdate emitted only when set.
- [x] 2.3 GREEN; gofmt/vet clean. Commit: `feat(claude): project marketplaces to extraKnownMarketplaces.<name>`

## 3. Regression and docs

- [x] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): a `[marketplaces.claude.official]` github marketplace + a `[plugins.claude.hud]` `source="hud@official"` → `apply` writes `extraKnownMarketplaces.official.source` and `enabledPlugins["hud@official"]`; second `plan` byte-identical.
- [x] 3.2 Update `docs/roadmap.md` (v1.2 COMPLETE: declare + enable/disable + config + marketplace) + README to show a `[marketplaces.claude.<name>]` example. No over-claim.
- [x] 3.3 Commit all changes.
