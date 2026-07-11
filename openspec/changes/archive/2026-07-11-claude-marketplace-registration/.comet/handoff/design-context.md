# Comet Design Handoff

- Change: claude-marketplace-registration
- Phase: design
- Mode: compact
- Context hash: 483eed54e91c6032df1bf86243fdfd574a1cc8e7299edf5984f265e52d5a00ea

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/claude-marketplace-registration/proposal.md

- Source: openspec/changes/claude-marketplace-registration/proposal.md
- Lines: 1-60
- SHA256: 19974ff1a54194f1f708124a973b86bcfbb07905ca1a960f1afdd5cdddbd581a

```md
## Why

The plugin declaration model (v1.2 #1/#2) projects Claude plugin enable/disable
(`enabledPlugins`) and per-plugin config (`pluginConfigs`), but a Claude plugin
is identified by `name@marketplace` and Claude only loads it if that marketplace
is registered in `extraKnownMarketplaces`. homonto has no way to declare a
marketplace, so a declared plugin from a custom marketplace can't actually
resolve. This change (v1.2 #3, the final plugin-configuration increment) adds a
marketplace declaration model and projects it to Claude's `extraKnownMarketplaces`.
OpenCode has no marketplace concept (its plugins are npm packages / local files),
so marketplaces are Claude-only.

## What Changes

- Add a marketplace declaration model: `[marketplaces.claude.<name>]` tables with
  a `source` type and its type-specific locator:
  - `source = "github"` → `repo = "owner/repo"`;
  - `source = "url"` → `url = "https://…"`;
  - `source = "git-subdir"` → `url = "…"`, `path = "…"`;
  - `source = "directory"` → `path = "./…"`;
  - optional `auto_update` (bool).
- **Validation**: the marketplace name is a valid key; `source` is one of the
  four recognized types; the locator field required by that type is present
  (github→repo, url→url, git-subdir→url+path, directory→path); unknown source
  types and missing locators are rejected naming the marketplace.
- **Claude adapter** gains a managed key namespace `marketplace.<name>` projecting
  to `extraKnownMarketplaces.<name>` in `settings.json`:
  - desired: `marketplace.<name>` = `{"source": {"source": <type>, <locator>…}[, "autoUpdate": <bool>]}`;
  - read-back: `extraKnownMarketplaces` members → `marketplace.<name>` (and
    `extraKnownMarketplaces` excluded from the generic settings read-back);
  - apply writes the object at `extraKnownMarketplaces.<name>`;
  - prune deletes `extraKnownMarketplaces.<name>` when de-declared;
  - adoption of a pre-existing entry works like other keys;
  - `marketplace.` added to the managed-prefix set.
- `settings.claude.extraKnownMarketplaces` added to the reserved-settings guard.
- Surgical + idempotent; unrelated keys and other marketplaces preserved.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-model`: adds the `[marketplaces.claude.<name>]` declaration model
  (source + type-specific locator + optional auto_update; validated).
- `tool-adapters`: the Claude adapter projects marketplaces to
  `extraKnownMarketplaces.<name>` (surgical, idempotent, pruned, adoptable).

## Impact

- `internal/config/config.go`: `Marketplace` type, `Marketplaces` table on
  `Config`, validation.
- `internal/adapter/claude/claude.go`: `marketplace.` namespace across
  desired/current/apply/prune; `util.go` managed prefix.
- Tests in `internal/config` and `internal/adapter/claude`.
- No new dependency. No OpenCode change (marketplaces are Claude-only).
- **Completes roadmap v1.2 Plugin Configuration** (declare + enable/disable +
  config + marketplace).

```

## openspec/changes/claude-marketplace-registration/design.md

- Source: openspec/changes/claude-marketplace-registration/design.md
- Lines: 1-77
- SHA256: 4d1f2e0c4441e415bc81eafc2a0e9b04d529ffffe80290554ea140807109e4c6

```md
## Context

Final v1.2 increment. Claude loads a `name@marketplace` plugin only if the
marketplace is registered in `extraKnownMarketplaces` (see the
`plugin-config-formats` research). This adds a `[marketplaces.claude.<name>]`
declaration model and a `marketplace.<name>` managed namespace in the Claude
adapter — structurally identical to the `pluginconfig.` namespace shipped in v1.2
#2, so the read-back-exclusion idempotency discipline applies again.

## Goals / Non-Goals

**Goals**: `[marketplaces.claude.<name>]` model (source type + locator +
`auto_update`); validation; project to `extraKnownMarketplaces.<name>`
(desired/read-back/apply/prune/adopt/deterministic).

**Non-Goals**: OpenCode marketplaces (none exist); remote fetching; validating
the repo/url actually resolves; auto-installing plugins.

## Decisions

### D1 — Model (`internal/config/config.go`)

```go
type Marketplace struct {
    Source     string `toml:"source"`      // github | url | git-subdir | directory
    Repo       string `toml:"repo"`        // github
    URL        string `toml:"url"`         // url, git-subdir
    Path       string `toml:"path"`        // git-subdir, directory
    AutoUpdate *bool  `toml:"auto_update"` // optional
}
type Marketplaces struct { Claude map[string]Marketplace `toml:"claude"` }
// Config gains: Marketplaces Marketplaces `toml:"marketplaces"`
```

### D2 — Validation

Per claude marketplace: `validateKey("marketplaces.claude", name)`; `source` must
be one of the four types; the type's required locator present:
github→`Repo`, url→`URL`, git-subdir→`URL`+`Path`, directory→`Path`. Unknown
type or missing locator → error naming the marketplace. Add
`extraKnownMarketplaces` to the `settings.claude` reserved-key rejection.

### D3 — Claude adapter `marketplace.<name>` namespace (mirror `pluginconfig.`)

- **desired()**: for each `name, mk := range c.Marketplaces.Claude`,
  `out["marketplace."+name] = mustJSON(marketplaceValue(mk))` where
  `marketplaceValue` builds `{"source": {"source": mk.Source, <locator>}, ["autoUpdate": *mk.AutoUpdate]}`.
  The `source` sub-object includes only the locator fields relevant to the type
  (github→`repo`; url→`url`; git-subdir→`url`,`path`; directory→`path`) so the
  desired shape is canonical and stable. `autoUpdate` is emitted only when set.
- **current()**: `for k, v := range objMembers(sj, "extraKnownMarketplaces") { out["marketplace."+k] = v }`, and add `"extraKnownMarketplaces"` to the generic settings read-back skip set (now: mcpServers, enabledPlugins, pluginConfigs, extraKnownMarketplaces).
- **apply write**: `case hasPrefix(c.Key, "marketplace."): SetJSON(sj, "extraKnownMarketplaces."+EscapePath(trim(c.Key,"marketplace.")), val)`.
- **prune**: `case hasPrefix(c.Key, "marketplace."): DeleteJSON(sj, "extraKnownMarketplaces."+EscapePath(...))`.
- **managed prefix** (`util.go`): add `"marketplace."`.
- Adoption: generic path, no special case.

The value stored/compared is the whole `extraKnownMarketplaces.<name>` object, so
read-back and desired agree → idempotent (same discipline as pluginconfig).

## Risks / Trade-offs

- **Four managed namespaces in settings.json now** (settings, enabledPlugins,
  pluginConfigs, extraKnownMarketplaces): the generic settings read-back must
  skip all four; a missed skip re-surfaces the object as a phantom `setting.`
  key and churns the plan. A dedicated test (marketplace + a setting + a plugin
  in one file, re-plan byte-identical) locks it in.
- **Canonical `source` sub-object**: only type-relevant locator fields are
  emitted, so a github marketplace never carries an empty `url`/`path` that would
  differ from an adopted on-disk entry. `jsonutil.Canonical` handles key order.

## Migration Plan

Additive; `[marketplaces]` is optional. No migration.

## Open Questions

None. This completes v1.2.

```

## openspec/changes/claude-marketplace-registration/tasks.md

- Source: openspec/changes/claude-marketplace-registration/tasks.md
- Lines: 1-17
- SHA256: 9dbdc7da2a2c3b3ba8727b085dff3f3a2286ee7a21112942398d9210039b5b1d

```md
## 1. Config model + validation (`internal/config`)

- [ ] 1.1 (TDD RED first) Add `Marketplace{Source,Repo,URL,Path string; AutoUpdate *bool}` (toml tags incl `auto_update`), `Marketplaces{Claude map[string]Marketplace}`, and `Config.Marketplaces` (`toml:"marketplaces"`).
- [ ] 1.2 (TDD RED first) Validation: per claude marketplace, `validateKey("marketplaces.claude", name)`; `source` ∈ {github,url,git-subdir,directory}; required locator present (github→repo, url→url, git-subdir→url+path, directory→path); unknown source or missing locator → error naming the marketplace. Add `extraKnownMarketplaces` to the `settings.claude` reserved-key rejection. Tests: parse github marketplace; unknown source rejected; missing repo (github) rejected; missing url/path (git-subdir) rejected; `settings.claude.extraKnownMarketplaces` rejected.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [marketplaces.claude.<name>] declaration model + validation`

## 2. Claude marketplace projection (`internal/adapter/claude`)

- [ ] 2.1 (TDD RED first) Add the `marketplace.` namespace (Design Doc D3): desired (`out["marketplace."+name] = mustJSON(marketplaceValue(mk))`, source sub-object with only type-relevant locator fields + optional autoUpdate); read-back (`objMembers(sj,"extraKnownMarketplaces")`→`marketplace.<k>` AND exclude `extraKnownMarketplaces` from the generic settings loop, now skipping mcpServers/enabledPlugins/pluginConfigs/extraKnownMarketplaces); apply write (`SetJSON extraKnownMarketplaces.<EscapePath(name)>`); prune (`DeleteJSON …`); `util.go` managed prefix `"marketplace."`.
- [ ] 2.2 (TDD RED first) Tests: github marketplace → `extraKnownMarketplaces[name].source == {"source":"github","repo":…}` on disk; de-declared marketplace pruned; adopt pre-existing matching entry; unrelated settings + other marketplaces preserved; consecutive plans byte-identical; a settings.json holding a setting + a plugin + a pluginConfig + a marketplace re-plans byte-identical (no namespace leaks); autoUpdate emitted only when set.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(claude): project marketplaces to extraKnownMarketplaces.<name>`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): a `[marketplaces.claude.official]` github marketplace + a `[plugins.claude.hud]` `source="hud@official"` → `apply` writes `extraKnownMarketplaces.official.source` and `enabledPlugins["hud@official"]`; second `plan` byte-identical.
- [ ] 3.2 Update `docs/roadmap.md` (v1.2 COMPLETE: declare + enable/disable + config + marketplace) + README to show a `[marketplaces.claude.<name>]` example. No over-claim.
- [ ] 3.3 Commit all changes.

```

## openspec/changes/claude-marketplace-registration/specs/config-model/spec.md

- Source: openspec/changes/claude-marketplace-registration/specs/config-model/spec.md
- Lines: 1-42
- SHA256: 8daa1c67c0ab8a1af9a24b491937f0e4abcd19c142cf806db8f5c0d45f4b4fc8

```md
## ADDED Requirements

### Requirement: Claude marketplace declaration model

Plugin marketplaces SHALL be declarable as `[marketplaces.claude.<name>]` tables.
Marketplaces are Claude-only (OpenCode has no marketplace concept). Each table
SHALL carry a `source` type and its type-specific locator:

- `source = "github"` requires `repo` (`"owner/repo"`);
- `source = "url"` requires `url`;
- `source = "git-subdir"` requires `url` and `path`;
- `source = "directory"` requires `path`;
- `auto_update` (optional boolean).

The marketplace name SHALL be validated as a config key. An unrecognized `source`
type, or a missing required locator field for the declared type, SHALL be
rejected at load naming the marketplace. `settings.claude.extraKnownMarketplaces`
SHALL be rejected as reserved (homonto manages that structure).

#### Scenario: Parse a github marketplace

- **GIVEN** `[marketplaces.claude.official]` with `source = "github"` and `repo = "anthropics/claude-plugins"`
- **WHEN** the config is parsed
- **THEN** it yields a Claude marketplace `official` with a github source and that repo

#### Scenario: Unknown source type is rejected

- **GIVEN** `[marketplaces.claude.x]` with `source = "svn"`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the marketplace and the invalid source

#### Scenario: Missing locator for the source type is rejected

- **GIVEN** `[marketplaces.claude.x]` with `source = "github"` and no `repo`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the marketplace and the missing field

#### Scenario: Reserved marketplace settings key rejected

- **GIVEN** a `settings.claude` key `extraKnownMarketplaces`
- **WHEN** the config is parsed
- **THEN** it is rejected as reserved

```

## openspec/changes/claude-marketplace-registration/specs/tool-adapters/spec.md

- Source: openspec/changes/claude-marketplace-registration/specs/tool-adapters/spec.md
- Lines: 1-36
- SHA256: e78c75933640e345f74bfa2d390f6ef2c838669349f776e9b73a7934b9805330

```md
## ADDED Requirements

### Requirement: Claude marketplace projection

The Claude adapter SHALL project declared `[marketplaces.claude.<name>]` entries
to `extraKnownMarketplaces.<name>` in `settings.json`, via a managed key
namespace `marketplace.<name>`, surgically and idempotently. Specifically:

- desired: each declared marketplace contributes `marketplace.<name>` whose value
  is `{"source": {"source": <type>, <locator fields>}[, "autoUpdate": <bool>]}`;
- read-back: existing `extraKnownMarketplaces` members are read back as
  `marketplace.<name>` and excluded from the generic settings read-back;
- apply: the object is written at `extraKnownMarketplaces.<name>`, preserving
  unrelated `settings.json` keys and other marketplaces;
- prune: a de-declared marketplace deletes `extraKnownMarketplaces.<name>`;
- adoption: a pre-existing `extraKnownMarketplaces.<name>` equal to the desired
  value is adopted into state without rewriting the file;
- consecutive plans are byte-identical (deterministic).

#### Scenario: github marketplace projected

- **GIVEN** `[marketplaces.claude.official]` (`source = "github"`, `repo = "anthropics/claude-plugins"`)
- **WHEN** apply runs
- **THEN** `settings.json` `extraKnownMarketplaces.official.source` is `{"source":"github","repo":"anthropics/claude-plugins"}`, and unrelated keys are preserved

#### Scenario: De-declared marketplace is pruned

- **GIVEN** an `extraKnownMarketplaces.<name>` previously written and recorded by homonto, whose marketplace is no longer declared
- **WHEN** apply runs
- **THEN** `extraKnownMarketplaces.<name>` is deleted from `settings.json`

#### Scenario: Marketplace plan is deterministic

- **GIVEN** a declared marketplace with an `auto_update` flag
- **WHEN** `plan` runs twice consecutively
- **THEN** the two plans are byte-identical

```
