# Crash-safe builtin-skill materialization (stage-then-swap)

## Why

Roadmap X2 (F47), catalog-materialization slice. `catalog.Catalog.Materialize`
writes each builtin skill by `os.RemoveAll(dstDir)` followed by a file-by-file
walk-write. If the walk fails partway — a read error, a full disk, or a process
crash — the skill directory is left **partially written**. The engine's
re-materialize gate `allSkillDirsExist` only `Stat`s the directory (checks it
exists and is a dir), so a partial skill dir passes the gate and is **never
repaired**, leaving a broken skill linked into the user's tools indefinitely.

Single-file commands and subagents already materialize atomically (each is one
`fsutil.WriteControlPlane`, a temp+rename). Only the multi-file per-skill
directory is destructive-before-complete.

## What Changes

Make per-skill materialization atomic: write the skill's files into a temporary
staging directory beside the destination, and only after the full walk succeeds
swap it into place (`RemoveAll(dst)` + `Rename(staging, dst)`). A failure mid-
walk leaves the previous complete skill dir untouched and discards the staging
dir; a crash in the tiny swap window leaves `dst` absent (not partial), so
`allSkillDirsExist` correctly re-materializes on the next run. A leftover
staging dir from a crash is removed before staging begins.

## Impact

- **Specs:** `apply-pipeline` gains a requirement that builtin-skill
  materialization is atomic per skill — a destination skill directory only ever
  contains a complete skill, never a partially-written one.
- **Behavior:** none on the success path — the materialized bytes are identical.
  The only change is crash/error safety.
- **Risk:** low — a localized change to one function, plus a new failure-path
  test; guarded by the existing catalog + engine materialization suites.

## Non-goals

- Staging for commands/subagents (already atomic single-file writes).
- A completion-marker or content-hash gate in `allSkillDirsExist` (the atomic
  swap makes directory presence a sufficient signal again).
- Broader X2 (stateless Apply, transaction journals, close/archive validation).
