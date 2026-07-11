---
change: opencode-tui-config
design-doc: docs/superpowers/specs/2026-07-11-opencode-tui-config-design.md
base-ref: 90d86a1aa37265b2bbfa13e04a7ab1563883a890
archived-with: 2026-07-11-opencode-tui-config
---

# Plan: OpenCode tui.json config (v1.3 #1)

Add `[tui.opencode]` → `~/.config/opencode/tui.json` as a SECOND managed file in
the opencode adapter (`tui.<key>` namespace, mirrors `setting.<key>` against a
second doc). See the Design Doc for exact edits + confirmed helper facts. TDD.

## Task 1: config model + validation (`internal/config`)

- [x] 1.1 (TDD RED first) Add `type TUI struct { OpenCode map[string]any `toml:"opencode"` }` + `Config.TUI TUI `toml:"tui"``.
- [x] 1.2 (TDD RED first) Validation: `for k := range c.TUI.OpenCode { validateKey("tui.opencode", k) }`. Tests: parse `[tui.opencode]` (theme + scroll_speed); index-like/empty key rejected.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [tui.opencode] TUI settings declaration`

## Task 2: OpenCode tui.json projection (`internal/adapter/opencode`)

- [x] 2.1 (TDD RED first) Design Doc D3: add `tuiFile()`; Plan reads `tuiDoc` + `planKey(st,"tui."+k,…)` per `c.TUI.OpenCode`; orphan-prune `declared["tui."+k]=true`; ObserveHashes reads `tuiDoc` + `tui.` case; Apply reads `tuiDoc`, `tui.` set+delete cases, `if tuiChanged { WriteAtomic(tuiFile(),tuiDoc) }`; add `"tui."` to `managedPrefix` (util.go).
- [x] 2.2 (TDD RED first) Tests: theme → tui.json.theme after apply, unrelated key preserved; de-declared pruned; adopt pre-existing (file untouched); tui-only config leaves opencode.jsonc byte-identical; settings+tui config re-plans byte-identical; missing tui.json → first apply creates it.
- [x] 2.3 GREEN; gofmt/vet clean. Commit: `feat(opencode): manage tui.json as a second config-file target`

## Task 3: Regression and docs

- [x] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): `[tui.opencode] theme="gruvbox"` + `[settings.opencode] model="x"` → apply writes tui.json + opencode.jsonc; second plan byte-identical.
- [x] 3.2 Update `docs/roadmap.md` v1.3 status + README `[tui.opencode]` example. No over-claim.
- [x] 3.3 Commit all changes.
