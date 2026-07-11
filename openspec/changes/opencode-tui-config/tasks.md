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
