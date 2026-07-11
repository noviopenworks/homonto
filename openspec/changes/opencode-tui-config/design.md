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
