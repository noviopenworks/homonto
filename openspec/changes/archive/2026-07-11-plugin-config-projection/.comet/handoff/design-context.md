# Comet Design Handoff

- Change: plugin-config-projection
- Phase: design
- Mode: compact
- Context hash: dd31d7d30f628831809ec46d1e3858c31a0e6f9e9b59775a0bcf54e4272f28a6

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/plugin-config-projection/proposal.md

- Source: openspec/changes/plugin-config-projection/proposal.md
- Lines: 1-56
- SHA256: 68764a42c892e9ab0bab7828c34d0d63029b20c4cba7b9fbd710e3d2bc1e3c6c

```md
## Why

The plugin declaration model (v1.2 #1) landed `[plugins.<tool>.<name>]` tables
with `source` + `enabled`, but not the `config` half of "declarations with
configuration." Claude Code stores non-sensitive per-plugin settings on disk as
`pluginConfigs.<plugin>.options`; homonto does not project there yet. This
change (v1.2 #2) adds a per-plugin `config` field and projects it to Claude
`pluginConfigs`. OpenCode has no native per-plugin config location (its plugins
are a plain `plugin` string array), so a `config` on an OpenCode plugin is
rejected at load rather than silently dropped — honoring the roadmap non-goal of
"no cross-tool abstraction that hides real Claude/OpenCode plugin differences."

## What Changes

- Add `Config map[string]any` (`toml:"config"`) to `internal/config`'s `Plugin`.
- **Validation**: an OpenCode plugin declaring a non-empty `config` is rejected
  at load with a clear message (OpenCode has no per-plugin config on disk).
  Claude plugins may declare `config`.
- **Claude adapter** gains a new managed key namespace `pluginconfig.<source>`
  that projects to `pluginConfigs.<source>.options` in `settings.json`:
  - desired: each Claude plugin with a non-empty `config` contributes
    `pluginconfig.<source> = {"options": <config>}`;
  - read-back: `pluginConfigs` members are read back as `pluginconfig.<key>`
    (and `pluginConfigs` is excluded from the generic settings read-back);
  - apply writes the `{options: …}` object at `pluginConfigs.<source>`;
  - prune deletes `pluginConfigs.<source>` when the config is de-declared;
  - adoption of a pre-existing `pluginConfigs.<source>` works like other keys;
  - `pluginconfig.` is added to the managed-prefix set.
- Surgical + idempotent: unrelated `settings.json` keys and other
  `pluginConfigs` entries are preserved; consecutive plans are byte-identical.
- `settings.claude.pluginConfigs` is added to the reserved-settings-keys guard
  (homonto now manages that structure).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-model`: the `Plugin` table gains an optional `config` map;
  OpenCode plugins with `config` are rejected.
- `tool-adapters`: the Claude adapter projects per-plugin `config` to
  `pluginConfigs.<source>.options` (surgical, idempotent, pruned, adoptable).

## Impact

- `internal/config/config.go`: `Plugin.Config`; OpenCode-config rejection;
  `settings.claude.pluginConfigs` reserved-key guard.
- `internal/adapter/claude/claude.go`: `pluginconfig.` namespace across
  desired/current/apply/prune; `internal/adapter/claude/util.go`: managed prefix.
- Tests in `internal/config` and `internal/adapter/claude`.
- No new dependency. No OpenCode adapter change (config rejected upstream).
- Deferred to a later increment: Claude marketplace registration
  (`extraKnownMarketplaces`), which needs a marketplace-declaration model.

```

## openspec/changes/plugin-config-projection/design.md

- Source: openspec/changes/plugin-config-projection/design.md
- Lines: 1-66
- SHA256: e209de62272b02a53b12c5f9f2d3cb96c249ac388b010a52fbaa1d8fb833abef

```md
## Context

v1.2 #1 landed the `[plugins.<tool>.<name>]` model with `source`/`enabled`. This
increment (#2) adds the `config` half. Claude stores per-plugin non-sensitive
settings at `pluginConfigs.<plugin>.options` (see the `plugin-config-formats`
research). OpenCode has no per-plugin config on disk, so OpenCode `config` is
rejected at load. Claude marketplace registration (`extraKnownMarketplaces`) is a
separate follow-up (needs a marketplace-declaration model).

## Goals / Non-Goals

**Goals**: `Plugin.Config map[string]any`; project Claude `config` →
`pluginConfigs.<source>.options` (new `pluginconfig.` managed namespace: desired,
read-back, apply, prune, adopt, deterministic); reject OpenCode `config` at load.

**Non-Goals**: Claude `extraKnownMarketplaces` (follow-up); OpenCode config
projection (no native home); secret-bearing plugin config values (config is
non-sensitive, same as `settings`).

## Decisions

### D1 — New Claude managed namespace `pluginconfig.<source>`

Mirror the existing `plugin.`/`setting.` key-namespace pattern in
`internal/adapter/claude/claude.go`:
- **desired** (`desired()`): `for _, pl := range c.Plugins.Claude { if len(pl.Config) > 0 { out["pluginconfig."+pl.Source] = mustJSON(map[string]any{"options": pl.Config}) } }`.
- **read-back** (`current()`): `for k, v := range objMembers(sj, "pluginConfigs") { out["pluginconfig."+k] = v }`, and add `"pluginConfigs"` to the skip set in the generic settings read-back loop.
- **apply write**: `case hasPrefix(c.Key, "pluginconfig."): sj = SetJSON(sj, "pluginConfigs."+EscapePath(trim(c.Key,"pluginconfig.")), val)`.
- **prune**: `case hasPrefix(c.Key, "pluginconfig."): sj = DeleteJSON(sj, "pluginConfigs."+EscapePath(trim(c.Key,"pluginconfig.")))`.
- **managed prefix** (`util.go`): add `"pluginconfig."` to the recognized prefix list.
- adoption is handled by the generic adopt path (records desired → hash of
  on-disk value) with no special case.

The value stored/compared is the whole `{"options": {...}}` object, so read-back
(`objMembers` yields each `pluginConfigs.<k>` member verbatim) and desired agree,
giving idempotent plans.

### D2 — OpenCode config rejected at load

In `config.go` validation, for each `opencode` plugin with `len(pl.Config) > 0`
return an error: OpenCode plugins are a plain array with no per-plugin config; do
not silently drop it. Claude plugins may carry `config`.

### D3 — Reserved key

Add `pluginConfigs` to the `settings.claude` reserved-key rejection (alongside
`enabledPlugins`, `mcpServers`).

## Risks / Trade-offs

- **Ordering with `enabledPlugins` and settings in the same file**: three managed
  namespaces (`enabledPlugins`, `pluginConfigs`, top-level settings) now coexist
  in `settings.json`; read-back must exclude BOTH `enabledPlugins` and
  `pluginConfigs` from the generic settings loop (or they'd double-count as
  `setting.` keys, breaking idempotency). Covered by a mixed test.
- **`config` value canonicalization**: `map[string]any` → `mustJSON` →
  `jsonutil.Canonical` for the state hash, same path as `setting` values, so
  key-order noise doesn't churn plans.

## Migration Plan

Additive within v1.2; `config` is optional. No migration.

## Open Questions

None. Marketplace registration is a deferred, scoped follow-up.

```

## openspec/changes/plugin-config-projection/tasks.md

- Source: openspec/changes/plugin-config-projection/tasks.md
- Lines: 1-17
- SHA256: 65688e3481531ef1eaa8832cecd5cfd356cbf4bf8ab872cbe31eadd23e6349f4

```md
## 1. Config model + validation (`internal/config`)

- [ ] 1.1 (TDD RED first) Add `Config map[string]any \`toml:"config"\`` to `Plugin`.
- [ ] 1.2 (TDD RED first) Validation: reject an `opencode` plugin with a non-empty `config` (error naming the plugin, explaining OpenCode has no per-plugin config); add `pluginConfigs` to the `settings.claude` reserved-key rejection. Tests: claude plugin with config parses; opencode plugin with config rejected; `settings.claude.pluginConfigs` rejected; existing enable/disable + dup-source guards still pass.
- [ ] 1.3 GREEN; gofmt/vet clean for `internal/config`. Commit: `feat(config): plugin config field + opencode-config/pluginConfigs guards`

## 2. Claude pluginConfigs projection (`internal/adapter/claude`)

- [ ] 2.1 (TDD RED first) Add the `pluginconfig.` managed namespace per the Design Doc: desired (`out["pluginconfig."+pl.Source] = mustJSON({"options": pl.Config})` for each claude plugin with non-empty config); read-back (`objMembers(sj,"pluginConfigs")` → `pluginconfig.<k>`, and exclude `pluginConfigs` from the generic settings read-back loop alongside `enabledPlugins`/`mcpServers`); apply write (`SetJSON pluginConfigs.<EscapePath(source)>`); prune (`DeleteJSON pluginConfigs.<...>`); add `"pluginconfig."` to `util.go` managed prefixes.
- [ ] 2.2 (TDD RED first) Tests: config projected → `pluginConfigs[source].options.<k>` on disk after apply; no config → no `pluginConfigs` entry; de-declared config pruned; adopt a pre-existing matching `pluginConfigs.<source>`; unrelated settings + other pluginConfigs entries preserved; consecutive plans byte-identical; a plugin with BOTH enabled+config projects `enabledPlugins[source]` AND `pluginConfigs[source].options` without either read-back double-counting as a `setting.`.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(claude): project per-plugin config to pluginConfigs.<source>.options`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E: a `homonto.toml` with a claude plugin carrying `config` → `homonto plan`/`apply` writes `pluginConfigs.<source>.options`; a second `plan` is byte-identical; an opencode plugin with `config` fails `homonto plan` with the rejection message.
- [ ] 3.2 Update `docs/roadmap.md` v1.2 status (per-plugin config projection landed; marketplace registration is the remaining v1.2 increment) + README `[plugins.claude.<name>]` example to show an optional `config`. No over-claim.
- [ ] 3.3 Commit all changes.

```

## openspec/changes/plugin-config-projection/specs/config-model/spec.md

- Source: openspec/changes/plugin-config-projection/specs/config-model/spec.md
- Lines: 1-62
- SHA256: 8d816dc7cdf29bbcae96632bde2bcbd1a247c8ab1b13654f43493ced8259ae68

```md
## MODIFIED Requirements

### Requirement: Plugin declaration model

Plugins SHALL be declared as per-tool, per-plugin tables
`[plugins.<tool>.<name>]` (tool ∈ {`claude`, `opencode`}). Each plugin table
SHALL carry:

- `source` (required, non-empty string): the tool-native plugin identifier —
  for `claude` the `name@marketplace` key used in `enabledPlugins`; for
  `opencode` the npm package name or local plugin path placed in the `plugin`
  array.
- `enabled` (optional boolean, default `true`): `false` marks the plugin
  disabled.
- `config` (optional table): non-sensitive per-plugin settings. `config` is
  supported only for `claude` plugins (projected to `pluginConfigs.<source>.options`);
  a non-empty `config` on an `opencode` plugin SHALL be rejected at load, because
  OpenCode has no per-plugin config location on disk.

The declaration name (the table key) and the `source` SHALL be validated with
the same key-validation guard applied to other config keys. A plugin whose
`source` is empty SHALL be rejected. Two plugin declarations under the same tool
sharing one `source` SHALL be rejected (their projections would collide). The
reserved-key guards SHALL be preserved and extended: `settings.claude.enabledPlugins`,
`settings.claude.pluginConfigs`, and `settings.opencode.plugin`/`mcp` remain
rejected because homonto manages those structures.

#### Scenario: Parse plugin declaration tables

- **GIVEN** a config with `[plugins.claude.claude-hud]` (`source = "claude-hud@official"`, `enabled = true`) and `[plugins.opencode.quota]` (`source = "@slkiser/opencode-quota"`, no `enabled`)
- **WHEN** the config is parsed
- **THEN** it yields a Claude plugin `claude-hud` with source `claude-hud@official` enabled, and an OpenCode plugin `quota` with source `@slkiser/opencode-quota` whose enabled defaults to true

#### Scenario: A plugin without a source is rejected

- **GIVEN** a `[plugins.claude.x]` table with no `source` (or an empty `source`)
- **WHEN** the config is parsed
- **THEN** it is rejected with an error identifying the plugin

#### Scenario: enabled defaults to true and false disables

- **GIVEN** one plugin with `enabled = false` and one with `enabled` omitted
- **WHEN** the config is parsed
- **THEN** the first is disabled and the second is enabled (default true)

#### Scenario: Reserved plugin settings keys still rejected

- **GIVEN** a `settings.claude` key `enabledPlugins` or `pluginConfigs`, or a `settings.opencode` key `plugin`
- **WHEN** the config is parsed
- **THEN** it is rejected as reserved (homonto manages plugins there)

#### Scenario: A Claude plugin config is parsed

- **GIVEN** a `[plugins.claude.hud]` with `source = "hud@official"` and `config = { api_endpoint = "https://x", max_workers = 4 }`
- **WHEN** the config is parsed
- **THEN** the plugin carries that config map

#### Scenario: An OpenCode plugin config is rejected

- **GIVEN** a `[plugins.opencode.q]` with `source = "q"` and a non-empty `config`
- **WHEN** the config is parsed
- **THEN** it is rejected with an error explaining OpenCode has no per-plugin config

```

## openspec/changes/plugin-config-projection/specs/tool-adapters/spec.md

- Source: openspec/changes/plugin-config-projection/specs/tool-adapters/spec.md
- Lines: 1-46
- SHA256: bdb21483ac5787deaaa508e8e4defe0199da9f2c44d1d213ac0c96184875f4ea

```md
## ADDED Requirements

### Requirement: Claude plugin config projection

The Claude adapter SHALL project a declared Claude plugin's `config` to
`pluginConfigs.<source>.options` in `settings.json`, via a managed key namespace
`pluginconfig.<source>`, surgically and idempotently. Specifically:

- desired state: each Claude plugin with a non-empty `config` contributes
  `pluginconfig.<source>` whose value is `{"options": <config>}`;
- read-back: existing `pluginConfigs` members are read back as
  `pluginconfig.<key>` and are excluded from the generic settings read-back;
- apply: the `{options: …}` object is written at `pluginConfigs.<source>`,
  preserving unrelated `settings.json` keys and other `pluginConfigs` entries;
- prune: a de-declared plugin config deletes `pluginConfigs.<source>`;
- adoption: a pre-existing `pluginConfigs.<source>` equal to the desired value is
  adopted into state without rewriting the file;
- consecutive plans are byte-identical (deterministic).

A Claude plugin without a `config` (or an empty one) contributes no
`pluginConfigs` entry. OpenCode has no per-plugin config projection (a `config`
on an OpenCode plugin is rejected at load).

#### Scenario: Claude plugin config projected under options

- **GIVEN** a Claude plugin `[plugins.claude.hud]` with `source = "hud@official"` and `config = { api_endpoint = "https://x" }`
- **WHEN** apply runs
- **THEN** `settings.json` `pluginConfigs["hud@official"].options.api_endpoint` is `"https://x"`, and unrelated keys are preserved

#### Scenario: Plugin without config projects no pluginConfigs entry

- **GIVEN** a Claude plugin with no `config`
- **WHEN** apply runs
- **THEN** no `pluginConfigs` entry is written for it

#### Scenario: De-declared plugin config is pruned

- **GIVEN** a `pluginConfigs.<source>` previously written and recorded by homonto, whose plugin no longer declares `config`
- **WHEN** apply runs
- **THEN** `pluginConfigs.<source>` is deleted from `settings.json`

#### Scenario: Plugin config plan is deterministic

- **GIVEN** a Claude plugin with a multi-key `config`
- **WHEN** `plan` runs twice consecutively
- **THEN** the two plans are byte-identical

```
