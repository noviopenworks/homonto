# Comet Design Handoff

- Change: plugin-declaration-model
- Phase: design
- Mode: compact
- Context hash: 334705d72412297064702763896d6c0e9f136dffa51081a5a9092635bec2e5e2

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/plugin-declaration-model/proposal.md

- Source: openspec/changes/plugin-declaration-model/proposal.md
- Lines: 1-76
- SHA256: e83ca7f71d37ff2b4ce24d2d428530086e016ea51db459845becf239adc4a9fb

```md
## Why

Roadmap v1.2 (Plugin Configuration) expands plugin support "from simple
references to declarations with configuration." Today `homonto.toml` declares
plugins as bare name lists — `[plugins] claude = ["x"]`, `opencode = ["y"]` —
and homonto can only *enable* a Claude plugin (`enabledPlugins.<name> = true`)
or *add* an OpenCode plugin (a `plugin` array entry). There is no way to
disable a plugin, or to bind a Claude plugin to its marketplace, or to carry
per-plugin config. This change lays the foundation: it replaces the bare-list
model with per-plugin declaration tables `[plugins.<tool>.<name>]` carrying a
`source` and an `enabled` flag, and teaches both adapters enable/disable
semantics. It is the first increment of v1.2; per-plugin `config` projection
(Claude `pluginConfigs`) and Claude marketplace registration
(`extraKnownMarketplaces`) are explicit follow-ups.

## What Changes

- **BREAKING (pre-release)**: the plugin config model changes from string lists
  to declaration tables. `internal/config`'s `Plugins{Claude []string;
  OpenCode []string}` becomes `Plugins{Claude map[string]Plugin; OpenCode
  map[string]Plugin}` with `type Plugin { Source string; Enabled *bool }`. In
  TOML:
  ```toml
  # before
  [plugins]
  claude = ["claude-hud"]
  opencode = ["opencode-quota"]
  # after
  [plugins.claude.claude-hud]
  source = "claude-hud@official"
  enabled = true
  [plugins.opencode.opencode-quota]
  source = "@slkiser/opencode-quota"
  # enabled defaults to true
  ```
  The map key is the plugin's declaration name; `source` is the tool-native
  identifier (Claude: the `name@marketplace` key used in `enabledPlugins`;
  OpenCode: the npm package / local plugin path placed in the `plugin` array).
  `enabled` is optional and defaults to `true`; `false` disables the plugin.
- **Validation**: each plugin's `source` is required (non-empty); the
  declaration name and source are validated as keys (same `validateKey` guard as
  today). The existing `settings.claude.enabledPlugins` and
  `settings.opencode.plugin`/`mcp` reserved-key guards are preserved.
- **Claude adapter**: projects `enabledPlugins[<source>] = <enabled>` for each
  declared Claude plugin — now writing `false` for a disabled plugin, not only
  `true`. Prune/adopt of `enabledPlugins.<key>` is unchanged.
- **OpenCode adapter**: an enabled plugin places its `source` in the `plugin`
  array (as today); a disabled plugin (`enabled = false`) is ensured absent
  (pruned if present). State is keyed by the declaration name.
- Existing plugin tests across `internal/config` and both adapters are updated
  to the new model.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `plugin-configuration`: plugins are declared as `[plugins.<tool>.<name>]`
  tables with a required `source` and an optional `enabled` flag (default true);
  homonto projects enable **and** disable to each tool's native plugin config.

## Impact

- `internal/config/config.go`: `Plugin` type; `Plugins` maps; validation.
- `internal/adapter/claude/claude.go`: `enabledPlugins` projection honors
  `source` + `enabled` (incl. `false`).
- `internal/adapter/opencode/opencode.go`: `plugin` array projection honors
  `source` + `enabled` (disabled → absent).
- Tests updated in `internal/config` and both adapters.
- **BREAKING** `homonto.toml` schema (pre-release, no migration shim); documented
  in the change. No new dependency.
- Follow-ups (NOT in scope): per-plugin `config` → Claude `pluginConfigs`;
  Claude `extraKnownMarketplaces` registration; OpenCode `config` handling.

```

## openspec/changes/plugin-declaration-model/design.md

- Source: openspec/changes/plugin-declaration-model/design.md
- Lines: 1-91
- SHA256: 819249787c4cefecd2b7ae14788679ab44384c932aa147e4ee24246b881ccda5

[TRUNCATED]

```md
## Context

Plugins are declared in `homonto.toml` as bare name lists and projected
minimally: Claude writes `enabledPlugins.<name> = true` (enable-only), OpenCode
appends `<name>` to the `plugin` array. Roadmap v1.2 expands this to
declarations with configuration. This change is v1.2's first increment: the
declaration-table model + enable/disable, deferring per-plugin `config` and
Claude marketplace registration to a follow-up.

The two tools' plugin systems genuinely differ (Claude: `enabledPlugins`
object keyed by `name@marketplace`, plus `extraKnownMarketplaces` and
`pluginConfigs`; OpenCode: a plain `plugin` string array with no per-plugin
config), so the model stays tool-scoped (`[plugins.claude.*]` vs
`[plugins.opencode.*]`) with tool-appropriate meaning for `source` — no unified
cross-tool abstraction (a stated roadmap non-goal).

## Goals / Non-Goals

**Goals**
- Declaration tables `[plugins.<tool>.<name>]` with required `source` +
  optional `enabled` (default true).
- Projectable disable (Claude `false`; OpenCode array removal), which the
  current model cannot express.
- Update both adapters + config validation + all existing plugin tests.

**Non-Goals (this increment)**
- Per-plugin `config` → Claude `pluginConfigs` (follow-up).
- Claude `extraKnownMarketplaces` registration (follow-up).
- OpenCode `config` handling (OpenCode has no native per-plugin config;
  follow-up decides warn/drop).
- Any migration shim for the old list form (pre-release breaking change).

## Decisions

### D1 — Model

`type Plugin struct { Source string \`toml:"source"\`; Enabled *bool
\`toml:"enabled"\` }`. `Plugins{ Claude map[string]Plugin \`toml:"claude"\`;
OpenCode map[string]Plugin \`toml:"opencode"\` }`. `Enabled` is a pointer so
"omitted" (nil → true) is distinguishable, though both nil and true mean
enabled. A helper `(Plugin).IsEnabled() bool` returns `Enabled == nil ||
*Enabled`.

### D2 — Validation

For each tool's plugins: `validateKey("plugins.<tool>", <declName>)` and reject
an empty `source`. Preserve the `settings.claude.enabledPlugins` and
`settings.opencode.plugin`/`mcp` reserved-key guards unchanged.

### D3 — Claude projection

In the desired-map builder, replace `out["plugin."+p] = \`true\`` with, for each
`name, pl := range c.Plugins.Claude`: `out["plugin."+name] =
mustJSON(pl.IsEnabled())` and project the value at `enabledPlugins[pl.Source]`.
The state/prune key stays `plugin.<name>`; the on-disk `enabledPlugins` key is
`pl.Source`. Disabled plugins now emit `false` (a real managed value) rather
than being absent — so `plan` shows disable, and apply writes it.

### D4 — OpenCode projection

For each `name, pl := range c.Plugins.OpenCode`: if `pl.IsEnabled()`, behave as
today but with the array value = `pl.Source` and state key `plugin.<name>`
(adopt/create as now). If `!pl.IsEnabled()`, ensure `pl.Source` is absent: if
present and managed in state, emit a prune (`delete`) change; else noop.

### D5 — Breaking change, no shim

The old `[plugins] claude = [...]` list no longer parses (the field is now a
map). Pre-release, acceptable. The change body documents the new form.

## Risks / Trade-offs

- **Decl-name vs source split** (state keyed by name, disk keyed by source):
  keeps state stable if a plugin is renamed at the source while the decl name is
  constant, and lets the OpenCode array hold the real package while state stays
  readable. Slight extra indirection in both adapters; covered by tests.
- **OpenCode disable = array removal** touches the shared array-membership prune
  path; must not remove entries homonto doesn't manage (guarded by state
  membership, like other prunes).
- **Test churn**: every existing plugin test moves from list to table form.

```

Full source: openspec/changes/plugin-declaration-model/design.md

## openspec/changes/plugin-declaration-model/tasks.md

- Source: openspec/changes/plugin-declaration-model/tasks.md
- Lines: 1-18
- SHA256: aaa4c59872039f11df2336dfb1ec77b8491af12584ffc1eca484eca9a1dde088

```md
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

```

## openspec/changes/plugin-declaration-model/specs/config-model/spec.md

- Source: openspec/changes/plugin-declaration-model/specs/config-model/spec.md
- Lines: 1-45
- SHA256: 1dc3a35cd42a60477984feeb1522bc88013cf8ddd272dd05b43da0fe41309a05

```md
## ADDED Requirements

### Requirement: Plugin declaration model

Plugins SHALL be declared as per-tool, per-plugin tables
`[plugins.<tool>.<name>]` (tool ∈ {`claude`, `opencode`}), replacing the prior
bare-name lists (`[plugins] claude = [...]`). This is a breaking, pre-release
schema change with no migration shim. Each plugin table SHALL carry:

- `source` (required, non-empty string): the tool-native plugin identifier —
  for `claude` the `name@marketplace` key used in `enabledPlugins`; for
  `opencode` the npm package name or local plugin path placed in the `plugin`
  array.
- `enabled` (optional boolean, default `true`): `false` marks the plugin
  disabled.

The declaration name (the table key) and the `source` SHALL be validated with
the same key-validation guard applied to other config keys. A plugin whose
`source` is empty SHALL be rejected. The existing reserved-key guards SHALL be
preserved: `settings.claude.enabledPlugins` and `settings.opencode.plugin` (and
`mcp`) remain rejected because homonto manages those structures.

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

- **GIVEN** a `settings.claude` key `enabledPlugins` or a `settings.opencode` key `plugin`
- **WHEN** the config is parsed
- **THEN** it is rejected as reserved (homonto manages plugins there)

```

## openspec/changes/plugin-declaration-model/specs/tool-adapters/spec.md

- Source: openspec/changes/plugin-declaration-model/specs/tool-adapters/spec.md
- Lines: 1-44
- SHA256: 6ea12171801037f994aca6529b255a7743f0058074a1c630b5ca0f04dd3a5960

```md
## ADDED Requirements

### Requirement: Plugin enable/disable projection

Both adapters SHALL project declared plugins from the
`[plugins.<tool>.<name>]` model, honoring each plugin's `source` and `enabled`
flag, surgically (unmanaged keys/entries preserved) and idempotently. A plugin's
projected state SHALL be keyed by its declaration name (`plugin.<name>`).

- **Claude**: for each declared Claude plugin, the adapter SHALL set
  `enabledPlugins[<source>]` to the plugin's `enabled` value in
  `settings.json` — writing `false` for a disabled plugin, not only `true`.
  Pruning and adoption of `enabledPlugins.<key>` are unchanged.
- **OpenCode**: an enabled plugin SHALL have its `source` present in the
  `plugin` array (appended without duplication); a disabled plugin
  (`enabled = false`) SHALL be ensured absent from the `plugin` array (pruned if
  present and managed).

Per-plugin `config` projection (Claude `pluginConfigs`) and Claude marketplace
registration (`extraKnownMarketplaces`) are OUT OF SCOPE for this requirement.

#### Scenario: Claude projects a disabled plugin as false

- **GIVEN** a Claude plugin `[plugins.claude.hud]` with `source = "hud@official"` and `enabled = false`
- **WHEN** apply runs
- **THEN** `settings.json` `enabledPlugins["hud@official"]` is `false`, and unrelated keys are preserved

#### Scenario: Claude projects an enabled plugin as true

- **GIVEN** a Claude plugin with `source = "hud@official"` and `enabled` omitted
- **WHEN** apply runs
- **THEN** `enabledPlugins["hud@official"]` is `true`

#### Scenario: OpenCode appends an enabled plugin's source

- **GIVEN** an OpenCode plugin `[plugins.opencode.quota]` with `source = "@slkiser/opencode-quota"` enabled, against an existing `plugin` array
- **WHEN** apply runs
- **THEN** `@slkiser/opencode-quota` is present in the `plugin` array without duplicating existing entries

#### Scenario: OpenCode removes a disabled plugin from the array

- **GIVEN** an OpenCode plugin declared `enabled = false` whose `source` is currently present in the `plugin` array as a homonto-managed entry
- **WHEN** apply runs
- **THEN** the `source` is removed from the `plugin` array

```
