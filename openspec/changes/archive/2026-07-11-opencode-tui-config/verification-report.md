# Verification Report: opencode-tui-config (v1.3 #1)

- **Change**: `opencode-tui-config` — `[tui.opencode]` → `~/.config/opencode/tui.json` (second managed file)
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (2 capabilities, config + adapter)
- **Result**: PASS — final review found no bugs

## Scope

`internal/config/config.go` (`TUI` type + `Config.TUI` + validation),
`internal/adapter/opencode/{opencode,util}.go` (a second managed file
`tui.json` with a `tui.<key>` namespace across Plan/ObserveHashes/Apply + prune +
managedPrefix), tests, README + roadmap. No Claude change (Claude TUI already
covered by `[settings.claude]` top-level settings.json keys).

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` decisions (D1 top-level `[tui.opencode]`, D3 second-file namespace) | PASS |
| 3 | Matches Design Doc (exact Plan/ObserveHashes/Apply edits, two-file independence) | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| parse `[tui.opencode]` | `TestLoadParsesTUIOpenCode` | PASS |
| invalid TUI key rejected | `TestLoadRejectsTUIIndexLikeName` | PASS |
| TUI setting → tui.json | `TestOpenCodeTUICreatesFileWithTheme` | PASS |
| unrelated tui.json key preserved | `TestOpenCodeTUIPreservesUnrelatedKey` | PASS |
| de-declared TUI key pruned | `TestOpenCodeTUIPrunesDeDeclaredKey` | PASS |
| adopt pre-existing (file untouched) | `TestOpenCodeTUIAdoptLeavesFileByteIdentical` | PASS |
| tui-only leaves opencode.jsonc byte-identical | `TestOpenCodeTUIOnlyLeavesOpencodeJsoncByteIdentical` | PASS |
| settings+tui idempotent | `TestOpenCodeTUIAndSettingsIdempotent` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 355 passed, 23 packages |
| `go test -race ./internal/config/... ./internal/adapter/opencode/...` | 91 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, temp $HOME)

A `[tui.opencode] theme="gruvbox" scroll_speed=3` + `[settings.opencode] model=…`
+ `[settings.claude] theme="dark"` config → `apply` wrote
`~/.config/opencode/tui.json` = `{scroll_speed:3, theme:gruvbox}`, `opencode.jsonc`
`model`, and `~/.claude/settings.json` top-level `theme:"dark"` (the already-
covered Claude path). A second `plan` reported **"No changes."** (idempotent
across all three targets / two OpenCode files).

## Code review (review_mode: standard) — no bugs

The final review verified all eight risk areas correct: two-file write
independence (`docChanged`/`tuiChanged` disjoint — a tui-only change leaves
`opencode.jsonc` byte-identical, JSONC comments preserved), `tuiDoc` read+error-
checked in all three methods, orphan-prune `declared["tui."+k]` + `"tui."` in
`managedPrefix` (idempotent + prunable), namespace disjointness, adopt/noop
state-only, EscapePath symmetry, and missing-file creation. No fixes required
(one MINOR non-blocking test-documentation nit).

## Conclusion

Verification PASS. First increment of roadmap v1.3, establishing the "two managed
files per adapter" pattern. Claude TUI needs no code (already covered by
`[settings.claude]`). Remaining v1.3: richer keybind/layout + theme-file handling
if wanted.
