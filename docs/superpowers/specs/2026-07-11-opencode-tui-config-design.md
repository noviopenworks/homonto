---
comet_change: opencode-tui-config
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-opencode-tui-config
status: final
---

# OpenCode tui.json Config — Technical Design

Deep refinement of `openspec/changes/opencode-tui-config/design.md`. Roadmap v1.3
#1: manage `~/.config/opencode/tui.json` as a SECOND config file in the OpenCode
adapter for `[tui.opencode]`. Formats: `tui-config-formats` memory. Verified
helper facts below.

## Confirmed helper behavior (targets are accurate)

- `readStandardized(path)` (opencode `util.go:52`) returns an EMPTY standardized
  doc on ENOENT — so a missing `tui.json` reads as `{}` and the first apply
  creates it. No special-casing needed.
- `fsutil.WriteAtomic(path, data)` (`fsutil.go:14`) does `os.MkdirAll(dir)`
  internally — writing `tui.json` creates `~/.config/opencode/` if absent. No
  separate mkdir.
- `managedPrefix(k)` (`util.go:43`) lists the prunable namespaces — MUST gain
  `"tui."`.
- The opencode.jsonc write is `if docChanged { fsutil.WriteAtomic(a.cfgFile(), doc) }`
  (`opencode.go:714`). Mirror it: `if tuiChanged { fsutil.WriteAtomic(a.tuiFile(), tuiDoc) }`.
- `planKey(st, key, want, disk, hasDisk)` produces create/update/noop/adopt — the
  exact `setting.` machinery; `tui.` reuses it verbatim.

## Model (`internal/config/config.go`)

```go
type TUI struct { OpenCode map[string]any `toml:"opencode"` }
// Config gains: TUI TUI `toml:"tui"`
```
No `[tui.claude]` (Claude TUI = `[settings.claude]` top-level settings.json keys,
already covered). Validation: `for k := range c.TUI.OpenCode { validateKey("tui.opencode", k) }`.

## OpenCode adapter (`internal/adapter/opencode/opencode.go`)

`func (a *Adapter) tuiFile() string { return filepath.Join(a.home, ".config", "opencode", "tui.json") }`

**Plan** (~244, after the `doc` read): add `tuiDoc, err := readStandardized(a.tuiFile())` (error-check). After the `[settings.opencode]` projection loop, add:
```go
for k, v := range c.TUI.OpenCode {
    key := "tui." + k
    want := mustJSON(v)
    disk, hasDisk := jsonutil.GetJSON(tuiDoc, jsonutil.EscapePath(k))
    cs.Changes = append(cs.Changes, planKey(st, key, want, disk, hasDisk))
}
```

**Orphan-prune declared set** (~416, where `declared["setting."+k]` etc. are built): add
```go
for k := range c.TUI.OpenCode { declared["tui."+k] = true }
```
so a de-declared `tui.` state key is pruned (the generic orphan loop at ~436 emits the delete; `managedPrefix` now recognizes `tui.`).

**ObserveHashes** (~496): add `tuiDoc, err := readStandardized(a.tuiFile())` and a case:
```go
case hasPrefix(key, "tui."):
    if v, ok := jsonutil.GetJSON(tuiDoc, jsonutil.EscapePath(trim(key, "tui."))); ok {
        out[key] = secret.Hash(jsonutil.Canonical(v))
    }
```

**Apply** (~574): add `tuiDoc, err := readStandardized(a.tuiFile())` and `tuiChanged := false`.
- delete switch (~613): `case hasPrefix(c.Key, "tui."): tuiDoc, err = jsonutil.DeleteJSON(tuiDoc, jsonutil.EscapePath(trim(c.Key, "tui."))); tuiChanged = true`.
- set switch (~682): `case hasPrefix(c.Key, "tui."): tuiDoc, err = jsonutil.SetJSON(tuiDoc, jsonutil.EscapePath(trim(c.Key, "tui.")), val); tuiChanged = true`.
- after the `if docChanged { WriteAtomic(cfgFile) }` block (~714): `if tuiChanged { if err := fsutil.WriteAtomic(a.tuiFile(), tuiDoc); err != nil { return err } }`.
adopt/noop remain state-only (no file write), same as settings — so an adopt never rewrites tui.json.

**managedPrefix** (`util.go:44`): add `"tui."` to the slice.

## Two-file independence (the one real risk)

Each file's write is gated by its own flag (`docChanged` / `tuiChanged`). A
config with only `[tui.opencode]` produces only `tui.` changes → `docChanged`
stays false → `opencode.jsonc` is NOT rewritten (JSONC comments preserved). A
config with only opencode.jsonc keys leaves `tui.json` untouched. Test both
directions.

## Tests

- config: parse `[tui.opencode]` (theme + scroll_speed); index-like/empty key rejected.
- opencode: `[tui.opencode] theme="gruvbox"` → after apply `tui.json.theme=="gruvbox"`, unrelated tui.json key preserved; de-declared key pruned from tui.json; adopt a pre-existing matching tui.json key (state-only, file bytes untouched); a config with ONLY `[tui.opencode]` leaves opencode.jsonc byte-identical; a config with `[settings.opencode]` + `[tui.opencode]` re-plans byte-identical (idempotent); missing tui.json → first apply creates it. Mirror the existing opencode setting tests' harness (temp home, Plan/Apply, read the file back).

## Verification

TDD RED→GREEN; full regression. E2E (real `homonto` binary): `[tui.opencode]
theme="gruvbox"` + `[settings.opencode] model="x"` → `apply` writes
`~/.config/opencode/tui.json {theme:gruvbox}` AND opencode.jsonc; second `plan`
byte-identical.

## Establishes

The "two managed files per adapter" pattern (future TUI/keybind increments, and
any tool that splits config across files).
