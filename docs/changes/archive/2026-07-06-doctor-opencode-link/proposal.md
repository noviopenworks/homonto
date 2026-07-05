Preset: fix

# Proposal: doctor-opencode-link

## Why

`homonto doctor` verifies each owned skill's content and its **Claude**
symlink, but never checks the **OpenCode** skill symlink
(`~/.config/opencode/skills/<name>`) — NEXT_AGENT gap #6, and a known gap
called out in the `cli-commands` spec. A broken or missing OpenCode link goes
unreported, so `doctor` can say a skill is fine when OpenCode can't see it.

## The bug (reproduction / expected vs actual)

- **GIVEN** an owned skill whose content exists and whose Claude link is
  correct but whose OpenCode link is missing or points elsewhere.
- **WHEN** `homonto doctor` runs.
- *Actual:* reports the skill `linked` (checks Claude only).
- *Expected:* reports `ok` for the Claude link and a `warn` for the missing
  OpenCode link, naming the tool.

## Fix scope

In `engine.Doctor` (`internal/engine/status.go`), check BOTH tool links per
owned skill (`~/.claude/skills/<name>` and
`~/.config/opencode/skills/<name>`), reporting `ok`/`warn` per tool. Update the
`cli-commands` doctor requirement (drop the "known gap" language). Tests.

## Capability Impact

- **Modified**: `cli-commands` — doctor now checks both tools' skill links
  (delta).
- Untouched: everything else.

## Grounding

`internal/engine/status.go` `Doctor` loops owned skills and readlinks only
`~/.claude/skills/<name>`. OpenCode links live at
`~/.config/opencode/skills/<name>` (opencode adapter `links()`). Spec gap:
`docs/specs/cli-commands.md` "doctor health checks" ("OpenCode skill symlink
checking is a known gap").

## Impact

- Files: `internal/engine/status.go`, `internal/engine/status_test.go`, delta
  `specs/cli-commands.md`. ≤5 non-test files — no upgrade trigger.
- Risk: output format changes (per-tool). Existing tests use substring matches
  (`linked` / `content present, not linked`) preserved, so they still pass.
