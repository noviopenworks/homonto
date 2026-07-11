# Brainstorm Summary
- Change: opencode-tui-config
- Date: 2026-07-11
## Confirmed Technical Approach
v1.3 #1. Top-level `[tui.opencode]` map → new second managed file `~/.config/opencode/tui.json` in the opencode adapter via a `tui.<key>` state namespace (mirrors `setting.<key>` but against tuiDoc). Plan/ObserveHashes/Apply each read a second doc; Apply gates a `tuiChanged` write. Orphan-prune declared set gains `tui.<key>`. `managedPrefix` gains `"tui."`. Reuses planKey/readStandardized/writeStandardized for surgical+idempotent+adopt. Claude TUI = already covered by `[settings.claude]` (no [tui.claude]). Formats: [[tui-config-formats]].
## Key Trade-offs and Risks
- Two-file Apply: each write gated by its own changed flag (all-tui change must NOT rewrite opencode.jsonc; test it byte-unchanged).
- Missing tui.json must read as empty doc (first apply creates it).
- Forgetting declared["tui."+k] would orphan-prune live keys every plan → idempotency test.
## Testing Strategy
TDD RED first. E2E: [tui.opencode] theme → tui.json on disk, idempotent, opencode.jsonc untouched by tui-only change. Full regression.
## Spec Patches
None. Delta specs (config-model + tool-adapters ADDED) carry the model + projection scenarios.
