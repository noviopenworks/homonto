# Comet Design Handoff

- Change: opencode-tui-config
- Phase: design
- Mode: compact
- Context hash: aa99c2d20f5f1b1e9e838c82bec6ae72e418d654115b3ee3e17a11ebbebd11c9

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/opencode-tui-config/proposal.md

- Source: openspec/changes/opencode-tui-config/proposal.md
- Lines: 1-60
- SHA256: 6cdd4357a812fd760499bbdb0b9ac5e30f29ab94619de6e43fc573050ea0af44

```md
## Why

Roadmap v1.3 (Tool TUI Configuration) manages Claude/OpenCode TUI-related config.
Research (see the `tui-config-formats` notes) shows Claude's TUI settings —
`theme`, `statusLine`, `tui`, `editorMode`, etc. — are all top-level
`settings.json` keys already covered by homonto's existing `[settings.claude]`
projection, so Claude needs no new code. OpenCode is different: its TUI settings
live in a **separate file** `~/.config/opencode/tui.json` (theme, keybinds,
scroll_speed, diff_style, mouse, …) that homonto does not currently write — it
only writes `opencode.jsonc`. This change adds an OpenCode `tui.json` projection
target: the first case of homonto managing **two** config files for one adapter.

## What Changes

- Add a top-level `[tui.opencode]` config table (a `map[string]any`) whose keys
  are written to `~/.config/opencode/tui.json`. A top-level `[tui.<tool>]` table
  (rather than nesting under `[settings.opencode]`, which specifically means
  `opencode.jsonc`) cleanly signals "the separate TUI file." Claude TUI settings
  intentionally have **no** `[tui.claude]` — they are top-level `settings.json`
  keys already handled by `[settings.claude]` (documented; avoids two ways to set
  one key). This diverges from the roadmap's illustrative `[settings.opencode.tui]`
  because TUI is a distinct file, so it earns a distinct table.
- **Validation**: each `[tui.opencode]` key is validated with the same key guard
  as other config keys.
- **OpenCode adapter** gains a second managed file `tui.json` and a `tui.<key>`
  state namespace, mirroring how `opencode.jsonc` settings are handled:
  - `Plan` reads `tui.json` and projects each `[tui.opencode]` key as
    `tui.<key>` (create/update/noop/adopt), and orphan-prunes de-declared
    `tui.<key>`;
  - `ObserveHashes` reads `tui.json` and hashes recorded `tui.<key>` values;
  - `Apply` writes `tui.json` (surgically; unmanaged keys preserved) only when a
    `tui.<key>` actually changed, and deletes a pruned key from it.
- Surgical + idempotent; unrelated `tui.json` keys preserved; consecutive plans
  byte-identical.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-model`: adds the `[tui.opencode]` declaration (map of TUI settings).
- `tool-adapters`: the OpenCode adapter manages a second config file,
  `~/.config/opencode/tui.json`, projecting `[tui.opencode]` keys (surgical,
  idempotent, pruned, adoptable).

## Impact

- `internal/config/config.go`: `TUI` type (`OpenCode map[string]any`), `Config.TUI`,
  validation.
- `internal/adapter/opencode/opencode.go`: a `tuiFile()` path + `tui.<key>`
  namespace across `Plan`, `ObserveHashes`, `Apply` (second-doc read/write) and
  the orphan-prune declared set.
- Tests in `internal/config` and `internal/adapter/opencode`.
- No new dependency. No Claude adapter change (Claude TUI already covered by
  `[settings.claude]`).
- Establishes the "two managed files per adapter" pattern for future TUI/keybind
  increments.

```

## openspec/changes/opencode-tui-config/design.md

- Source: openspec/changes/opencode-tui-config/design.md
- Lines: 1-80
- SHA256: a5c65835877ce61a338ba78bab244bb0d920c3df74c4771018b83f62d194bfdf

```md
## Context

Roadmap v1.3. Research (`tui-config-formats`): Claude TUI settings are top-level
`settings.json` keys already covered by `[settings.claude]`; OpenCode TUI settings
live in a SEPARATE file `~/.config/opencode/tui.json`. This change adds that file
as a second managed target in the OpenCode adapter — the first "two managed files
per adapter" case. The `tui.<key>` namespace mirrors the existing `setting.<key>`
handling but against `tui.json` instead of `opencode.jsonc`.

## Goals / Non-Goals

**Goals**: `[tui.opencode]` model + validation; project it to `tui.json`
(Plan/ObserveHashes/Apply + prune + adopt, surgical, idempotent).

**Non-Goals**: Claude TUI (already covered — no `[tui.claude]`); custom
theme-file management (`~/.claude/themes/*`, `~/.config/opencode/themes/*`);
keybind-specific merge semantics (a keybind is just a `keybinds` object value
here — projected verbatim); validating theme names.

## Decisions

### D1 — Model: top-level `[tui.opencode]`

```go
type TUI struct { OpenCode map[string]any `toml:"opencode"` }
// Config gains: TUI TUI `toml:"tui"`
```
A top-level `[tui.<tool>]` table, not `[settings.opencode.tui]` — `[settings.opencode]`
means `opencode.jsonc`, and `tui.json` is a distinct file, so it earns a distinct
table. No `[tui.claude]` (Claude TUI = `[settings.claude]` top-level keys). Diverges
from the roadmap's illustrative `[settings.opencode.tui]`, justified by the
separate-file reality.

### D2 — Validation

For each `[tui.opencode]` key: `validateKey("tui.opencode", key)` (same index-like/
empty-name guard as settings). No reserved keys.

### D3 — Second managed file in the OpenCode adapter

- `func (a *Adapter) tuiFile() string { return filepath.Join(a.home, ".config", "opencode", "tui.json") }`.
- **Plan** (`opencode.go` ~244): after the opencode.jsonc `doc` read, also
  `tuiDoc, err := readStandardized(a.tuiFile())`. For each `k, v := range c.TUI.OpenCode`:
  `key := "tui."+k; want := mustJSON(v); disk, hasDisk := jsonutil.GetJSON(tuiDoc, jsonutil.EscapePath(k)); cs.Changes = append(cs.Changes, planKey(st, key, want, disk, hasDisk))` — exactly the `setting.` pattern but against `tuiDoc` and the `tui.` prefix.
- **Orphan-prune** (~416): add `for k := range c.TUI.OpenCode { declared["tui."+k] = true }` so de-declared `tui.` keys are pruned (the generic orphan loop emits the delete for any managed `tui.` state key not in `declared`).
- **ObserveHashes** (~496): read `tuiDoc`; add `case hasPrefix(key, "tui."): if v, ok := jsonutil.GetJSON(tuiDoc, jsonutil.EscapePath(trim(key, "tui."))); ok { out[key] = secret.Hash(jsonutil.Canonical(v)) }`.
- **Apply** (~574): read `tuiDoc`; add a `tuiChanged` flag. In the delete switch:
  `case hasPrefix(c.Key, "tui."): tuiDoc, err = jsonutil.DeleteJSON(tuiDoc, jsonutil.EscapePath(trim(c.Key, "tui."))); tuiChanged = true`. In the set switch:
  `case hasPrefix(c.Key, "tui."): tuiDoc, err = jsonutil.SetJSON(tuiDoc, jsonutil.EscapePath(trim(c.Key, "tui."))), val); tuiChanged = true`. After the existing opencode.jsonc write, add a symmetric `if tuiChanged { writeStandardized(a.tuiFile(), tuiDoc) }` (mirror the exact write+mkdir the jsonc path uses). adopt/noop stay state-only (no file write), same as opencode.jsonc.
- `managedPrefix`/`managedKey` (whatever the opencode orphan-prune uses) must
  recognize `tui.` — check the existing `managedPrefix` helper and add `"tui."`.

### D4 — Reuse `planKey` / standardized read-write

`tui.json` is plain JSON; `readStandardized`/`writeStandardized` and `planKey`
already give surgical merge, adopt, and deterministic canonical hashing — the
same machinery `setting.` uses. So `tui.` inherits idempotency, adoption, and
JSONC-safe writes for free.

## Risks / Trade-offs

- **Two-file Apply**: Apply now conditionally writes two files. Each write is
  gated by its own `changed` flag, so an all-`tui` change never rewrites
  opencode.jsonc and vice-versa (keeps JSONC comments intact). Test: a config
  with only `[tui.opencode]` leaves opencode.jsonc byte-unchanged.
- **Missing tui.json**: `readStandardized` on a non-existent file must yield an
  empty doc (same as opencode.jsonc first-run) so the first apply creates it.
  Confirm `readStandardized` handles absence; if it errors on ENOENT, treat
  missing as empty.
- **Prune declared set**: forgetting `declared["tui."+k]` would orphan-prune a
  live TUI key every plan. Locked in by an idempotency test.

## Migration Plan

Additive; `[tui.opencode]` optional. No migration.

## Open Questions

None. Keybind-object and theme-file management are out of scope (future
increments if wanted).

```

## openspec/changes/opencode-tui-config/tasks.md

- Source: openspec/changes/opencode-tui-config/tasks.md
- Lines: 1-17
- SHA256: aa1ef9dc11c7fa1e49866748782e794f8c6a5ede03ff8af14427f1c62f7ea2bf

```md
## 1. Config model + validation (`internal/config`)

- [ ] 1.1 (TDD RED first) Add `type TUI struct { OpenCode map[string]any \`toml:"opencode"\` }` and `Config.TUI TUI \`toml:"tui"\``.
- [ ] 1.2 (TDD RED first) Validation: for each `c.TUI.OpenCode` key, `validateKey("tui.opencode", key)`. Tests: parse `[tui.opencode]` (theme + scroll_speed); an index-like/empty key rejected naming it.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [tui.opencode] TUI settings declaration`

## 2. OpenCode tui.json projection (`internal/adapter/opencode`)

- [ ] 2.1 (TDD RED first) Add `tuiFile()` = `~/.config/opencode/tui.json`; wire the `tui.<key>` namespace per Design Doc D3: Plan reads `tuiDoc` and `planKey(st,"tui."+k,want,disk,hasDisk)` for each `c.TUI.OpenCode` key; orphan-prune `declared["tui."+k]=true`; ObserveHashes reads `tuiDoc` + a `tui.` case; Apply reads `tuiDoc`, `tui.` cases in the set+delete switches, and a symmetric `if tuiChanged { write tui.json }` (mkdir like the jsonc path). Add `"tui."` to the opencode managed-prefix helper. Confirm `readStandardized` treats a missing tui.json as an empty doc (first apply creates it).
- [ ] 2.2 (TDD RED first) Tests: `[tui.opencode] theme="gruvbox"` → after apply, `tui.json` `theme=="gruvbox"`, an unrelated tui.json key preserved; de-declared key pruned from tui.json; adopt a pre-existing matching tui.json key (state-only, file untouched); a config with ONLY `[tui.opencode]` leaves opencode.jsonc byte-unchanged; a config with both `[settings.opencode]` and `[tui.opencode]` re-plans byte-identical (idempotent, no cross-file leak); missing tui.json → first apply creates it.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(opencode): manage tui.json as a second config-file target`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): `[tui.opencode] theme="gruvbox"` + `[settings.opencode] model="x"` → `apply` writes `~/.config/opencode/tui.json` `{theme:gruvbox}` AND `opencode.jsonc`; second `plan` byte-identical; a claude `[settings.claude] theme="dark"` still projects to settings.json (already-covered, unchanged).
- [ ] 3.2 Update `docs/roadmap.md` v1.3 status (OpenCode tui.json target landed; Claude TUI already covered by `[settings.claude]`) + README to show a `[tui.opencode]` example. No over-claim.
- [ ] 3.3 Commit all changes.

```

## openspec/changes/opencode-tui-config/specs/config-model/spec.md

- Source: openspec/changes/opencode-tui-config/specs/config-model/spec.md
- Lines: 1-22
- SHA256: 165d79d52550812dd3ac78c193814e31d7ec21bd8f6ddc9f7f91d3603aeee421

```md
## ADDED Requirements

### Requirement: OpenCode TUI settings declaration

OpenCode TUI settings SHALL be declarable as a top-level `[tui.opencode]` table
(a map of key → value) whose entries homonto projects to
`~/.config/opencode/tui.json`. Each `[tui.opencode]` key SHALL be validated with
the same key-validation guard applied to other config keys. Claude TUI settings
SHALL NOT have a `[tui.claude]` table — they are ordinary top-level
`settings.json` keys already declarable under `[settings.claude]`.

#### Scenario: Parse OpenCode TUI settings

- **GIVEN** a config with `[tui.opencode]` containing `theme = "gruvbox"` and `scroll_speed = 3`
- **WHEN** the config is parsed
- **THEN** it yields an OpenCode TUI settings map with those two entries

#### Scenario: Invalid TUI key is rejected

- **GIVEN** a `[tui.opencode]` key that is an index-like or empty name (invalid config key)
- **WHEN** the config is parsed
- **THEN** it is rejected naming the key

```

## openspec/changes/opencode-tui-config/specs/tool-adapters/spec.md

- Source: openspec/changes/opencode-tui-config/specs/tool-adapters/spec.md
- Lines: 1-45
- SHA256: ff40266bb5d9477b2905de257c18d2e174fb5581ce5ec25709726d6f363e918b

```md
## ADDED Requirements

### Requirement: OpenCode TUI file projection

The OpenCode adapter SHALL manage a second config file
`~/.config/opencode/tui.json` (in addition to `opencode.jsonc`), projecting each
`[tui.opencode]` key to a top-level key in `tui.json` via a `tui.<key>` state
namespace, surgically and idempotently. Specifically:

- `Plan` SHALL read `tui.json` and, for each declared `[tui.opencode]` key,
  produce a `tui.<key>` change (create / update / noop / adopt of a pre-existing
  matching key); a de-declared `tui.<key>` recorded in state SHALL be pruned;
- `ObserveHashes` SHALL read `tui.json` and hash the current value of each
  recorded `tui.<key>` still present;
- `Apply` SHALL write `tui.json` only when a `tui.<key>` change is applied,
  preserving unmanaged keys in the file, and SHALL delete a pruned key from it;
- unmanaged `tui.json` keys SHALL be preserved and consecutive plans SHALL be
  byte-identical.

Managing `tui.json` SHALL NOT alter how `opencode.jsonc` (MCPs, settings,
plugins) is managed.

#### Scenario: OpenCode TUI setting projected to tui.json

- **GIVEN** `[tui.opencode]` with `theme = "gruvbox"` against a `tui.json` with an unrelated key
- **WHEN** apply runs
- **THEN** `tui.json` `theme` is `"gruvbox"` and the unrelated key is preserved

#### Scenario: De-declared TUI setting is pruned from tui.json

- **GIVEN** a `tui.json` `theme` previously written and recorded by homonto, no longer declared
- **WHEN** apply runs
- **THEN** `theme` is deleted from `tui.json`

#### Scenario: TUI projection is idempotent and independent of opencode.jsonc

- **GIVEN** a config with both `[settings.opencode]` (→ opencode.jsonc) and `[tui.opencode]` (→ tui.json) keys, already applied
- **WHEN** `plan` runs twice consecutively
- **THEN** both plans are byte-identical and report no changes

#### Scenario: Adopt a pre-existing matching tui.json key

- **GIVEN** a `tui.json` already containing `theme = "gruvbox"` equal to the declared value, unrecorded in state
- **WHEN** apply runs
- **THEN** the key is adopted into state without rewriting `tui.json`

```
