---
change: claude-marketplace-registration
design-doc: docs/superpowers/specs/2026-07-11-claude-marketplace-registration-design.md
base-ref: f8d47f9b411f375530b55256e551a6690669cd58
---

# Plan: claude marketplace registration (v1.2 #3, final)

Add `[marketplaces.claude.<name>]` model + project to
`extraKnownMarketplaces.<name>` via a new `marketplace.` managed namespace
(mirrors `pluginconfig.`). See the Design Doc for the exact Go edits, the
`marketplaceValue` helper, and the read-back-exclusion hazard (now FOUR
namespaces). TDD.

## Task 1: config model + validation (`internal/config`)

- [x] 1.1 (TDD RED first) Add `Marketplace{Source,Repo,URL,Path string; AutoUpdate *bool}` (toml tags incl `auto_update`), `Marketplaces{Claude map[string]Marketplace}`, `Config.Marketplaces`.
- [x] 1.2 (TDD RED first) Validation loop: `validateKey("marketplaces.claude", name)`; source ∈ {github,url,git-subdir,directory} with required locator (github→repo, url→url, git-subdir→url+path, directory→path); unknown source / missing locator → error naming the marketplace. Add `extraKnownMarketplaces` to `settings.claude` reserved keys. Tests: parse github; unknown source rejected; missing repo rejected; missing url/path (git-subdir) rejected; reserved key rejected.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [marketplaces.claude.<name>] declaration model + validation`

## Task 2: Claude marketplace projection (`internal/adapter/claude`)

- [x] 2.1 (TDD RED first) Add the `marketplace.` namespace (Design Doc D3): `marketplaceValue(mk)` helper (canonical source sub-object, only type-relevant locator fields + optional autoUpdate); desired; read-back (`objMembers(sj,"extraKnownMarketplaces")`→`marketplace.<k>` AND skip `extraKnownMarketplaces` in the generic settings loop — now four: mcpServers/enabledPlugins/pluginConfigs/extraKnownMarketplaces); apply write (`SetJSON extraKnownMarketplaces.<EscapePath(name)>`); prune; `util.go` managed prefix `"marketplace."`.
- [x] 2.2 (TDD RED first) Tests: github marketplace → `extraKnownMarketplaces[name].source=={"source":"github","repo":…}` after apply; url/directory/git-subdir locator shapes; autoUpdate only when set; de-declared pruned; adopt pre-existing; unrelated settings + other marketplaces preserved; consecutive plans byte-identical; a settings.json with setting+plugin+pluginConfig+marketplace re-plans byte-identical (no namespace leaks).
- [x] 2.3 GREEN; gofmt/vet clean. Commit: `feat(claude): project marketplaces to extraKnownMarketplaces.<name>`

## Task 3: Regression and docs

- [x] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): `[marketplaces.claude.official]` github + `[plugins.claude.hud] source="hud@official"` → `apply` writes `extraKnownMarketplaces.official.source` + `enabledPlugins["hud@official"]`; second `plan` byte-identical.
- [x] 3.2 Update `docs/roadmap.md` (v1.2 COMPLETE: declare + enable/disable + config + marketplace) + README `[marketplaces.claude.<name>]` example. No over-claim.
- [x] 3.3 Commit all changes.
