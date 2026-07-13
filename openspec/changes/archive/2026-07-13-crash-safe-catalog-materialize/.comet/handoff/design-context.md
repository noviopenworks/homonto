# Comet Design Handoff

- Change: crash-safe-catalog-materialize
- Phase: design
- Mode: compact
- Context hash: cea8fec6a8c8551d4cb8ec359a8b03d00c4d586b8bcacd4fb42184121a3c29cc

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/crash-safe-catalog-materialize/proposal.md

- Source: openspec/changes/crash-safe-catalog-materialize/proposal.md
- Lines: 1-42
- SHA256: 3f4af323c9288f264cb858f22e5c894a659dc0678409302e4e9e7b810b6ad22a

```md
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

```

## openspec/changes/crash-safe-catalog-materialize/design.md

- Source: openspec/changes/crash-safe-catalog-materialize/design.md
- Lines: 1-35
- SHA256: 1fedf828d45253d0a17a2f8831e39f527046bcd330252ac958ebfbc08e36eb2b

```md
# Design — crash-safe catalog materialization

## Approach

In `catalog.Catalog.Materialize`, per skill:
1. `staging := dstDir + ".staging"`; `os.RemoveAll(staging)` first (discard any
   leftover from a prior crash).
2. Walk the embedded skill FS writing into `staging` (same `WriteControlPlane`
   no-follow writes as today, just rooted at `staging`).
3. On walk success: `os.RemoveAll(dstDir)` then `os.Rename(staging, dstDir)`.
4. On any walk error: return it; `staging` is left for the next run's step-1
   cleanup, and `dstDir` is untouched (still the prior complete version).

Commands/subagents are unchanged (already atomic single-file writes).

## Why this is sufficient

- Mid-walk failure → `dstDir` intact (old complete version), no partial dst.
- Crash in the RemoveAll→Rename window → `dstDir` absent (not partial), so
  `allSkillDirsExist` returns false and re-materializes next run.
- `Rename` within the same `.homonto` parent is atomic on POSIX.
- No change to the success-path bytes, so all existing materialize tests pass.

## Risk / safety

Localized to one function. The staging dir lives under the same control-plane
root as the destination, so `WriteControlPlane`'s no-follow guarantee still
holds. New test drives a mid-walk failure (an unreadable/oversized entry or an
injected error) and asserts the destination is never partial.

## Alternatives

- Completion marker / content-hash in `allSkillDirsExist` — rejected; atomic
  swap makes directory presence a sufficient completeness signal without a
  second gate to keep in sync.

```

## openspec/changes/crash-safe-catalog-materialize/tasks.md

- Source: openspec/changes/crash-safe-catalog-materialize/tasks.md
- Lines: 1-11
- SHA256: 90b685f31b269cd06a59e5dec6f92cee25b9c4c3f04ac10f921ee005cec291c3

```md
# Tasks — crash-safe-catalog-materialize

## 1. Stage-then-swap skill materialization
- [ ] catalog.Materialize writes each skill into a staging dir, then atomically
      swaps it into place (remove leftover staging first; RemoveAll(dst)+Rename
      on success). TDD: a walk that fails mid-skill leaves the prior dst intact
      and no partial dst; success writes identical bytes.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      existing catalog + engine materialize suites pass unchanged.

```

## openspec/changes/crash-safe-catalog-materialize/specs/apply-pipeline/spec.md

- Source: openspec/changes/crash-safe-catalog-materialize/specs/apply-pipeline/spec.md
- Lines: 1-25
- SHA256: aecba1ac87e9c86389c6625ca1a86fc6327f161e61fc9e4d3c30ffa6b5d4d84a

```md
# apply-pipeline

## ADDED Requirements

### Requirement: Builtin-skill materialization is atomic per skill

Materializing a builtin skill's directory SHALL be atomic: the destination skill
directory MUST only ever contain a complete skill, never a partially-written
one. Implementations MUST write the skill's files to a staging location and swap
it into place only after all files are written, so that a read error, full disk,
or process crash during materialization leaves either the previous complete skill
directory or no directory at all — never a partial one that the re-materialize
gate would mistake for a complete skill.

#### Scenario: A failure mid-materialization does not corrupt the destination

- **WHEN** materializing a skill fails partway through writing its files
- **THEN** the destination skill directory is left in its prior complete state
  (or absent if it never existed), never partially written

#### Scenario: Successful materialization writes identical content

- **WHEN** a skill materializes successfully via stage-then-swap
- **THEN** the destination contains exactly the skill's files, byte-for-byte the
  same as a direct write

```
