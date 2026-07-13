# Comet Design Handoff

- Change: close-archive-rollback
- Phase: design
- Mode: compact
- Context hash: 67b401771ce2968f1740bc413b38e04a37c3bb90285b146170d45cff7733017d

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/close-archive-rollback/proposal.md

- Source: openspec/changes/close-archive-rollback/proposal.md
- Lines: 1-39
- SHA256: 803e0c2d8980b965d9ba690ee0f4f6de7595a2c9b695c90674acbaaf999133f1

```md
# onto close: leave nothing archived when the archive move fails

## Why

Roadmap X2 (F4). The `onto close` spec says every failing case "archives
NOTHING", but the implementation sets `archived: true` and saves
`onto-state.yaml` **before** the `os.Rename` that moves the change into
`docs/changes/archive/`. If the rename fails (e.g. a permission error on the
archive parent, or a cross-device boundary), the change is left with
`archived: true` in state but still sitting at its original path — a
marked-archived-but-not-moved inconsistency that contradicts the spec's
"archives nothing on failure" guarantee and confuses later `onto` commands.

## What Changes

Make the two-step mutation consistent on the error path: if the archive move
fails, roll the `archived` flag back to `false` (and re-save state) so a failed
close leaves the change fully un-archived — exactly the "archives NOTHING"
contract. On success the behavior is unchanged (`archived: true`, moved,
reported). The state file lives inside the change dir, so a successful rename
carries the `archived: true` record into the archive location as before; the
rollback path re-saves the still-present in-place state file.

## Impact

- **Specs:** the `onto close archives a completed change` requirement is
  clarified to state that a failed archive move leaves `archived` unset.
- **Behavior:** only the failure path changes — a rename failure now leaves
  `archived: false` instead of a stale `true`. Success path unchanged.
- **Risk:** low — a localized error-path rollback in `runClose`, covered by a
  new failure-injection test (rename made to fail) plus the existing onto close
  suite.

## Non-goals

- Full crash-safety of the flag+move (a process kill between the save and the
  rename still leaves a window — closing that needs location-derived archived
  state, a larger change). This fixes the deterministic error path.
- Broader X2 (stateless Apply, transaction journals).

```

## openspec/changes/close-archive-rollback/design.md

- Source: openspec/changes/close-archive-rollback/design.md
- Lines: 1-41
- SHA256: 10337e74952281d7974e5f62335d2df691d74cd5cb204e396683b83754a5e6a0

```md
# Design — onto close archive-move rollback

## Approach

In `internal/ontocli/close.go` `runClose`, the tail is:
```
st.Archived = true
Save(statePath, st)
MkdirAll(archive parent)
Rename(changeDir, archiveDir)
```
Wrap the destructive steps so a failure after marking `archived` restores
consistency:
```
st.Archived = true
if err := Save(statePath, st); err != nil { return err }
rollback := func() { st.Archived = false; _ = Save(statePath, st) }
if err := MkdirAll(archive parent); err != nil { rollback(); return err }
if err := Rename(changeDir, archiveDir); err != nil { rollback(); return err }
```
On the success path nothing changes: the rename moves the change dir (with its
`archived: true` state file) into the archive location.

## Why save-then-rollback rather than rename-first

The `archived` flag lives in `onto-state.yaml` inside the change dir. Saving it
first keeps the on-success record co-located so the atomic rename carries it
into the archive with no second write at the new path. The only cost is the
error-path rollback, which this change adds. (Deriving archived-ness from the
directory's location instead of a flag would remove the window entirely but is a
larger redesign — out of scope.)

## Test

Force `Rename` to fail deterministically (e.g. pre-create the dated archive
target as a file, or make the archive parent unwritable) and assert: `runClose`
returns an error AND the change's `onto-state.yaml` still has `archived: false`
(rolled back) and the change dir is still at its original path.

## Alternatives
- Location-derived archived state — rejected as too large for this slice.

```

## openspec/changes/close-archive-rollback/tasks.md

- Source: openspec/changes/close-archive-rollback/tasks.md
- Lines: 1-10
- SHA256: 045b39ca8c8e1f4f7ab173fd4078e933fd2da3ce1c651aecb905aceacc5c12f2

```md
# Tasks — close-archive-rollback

## 1. Roll back archived flag on move failure
- [ ] runClose rolls st.Archived back to false (re-save) if MkdirAll/Rename
      fails. TDD: an injected rename failure leaves archived=false and the
      change unmoved; success path unchanged.

## 2. Verify
- [ ] `go test ./internal/ontocli/... -race`, vet, build, `openspec validate
      --all` green.

```

## openspec/changes/close-archive-rollback/specs/onto-binary/spec.md

- Source: openspec/changes/close-archive-rollback/specs/onto-binary/spec.md
- Lines: 1-46
- SHA256: ab44328e144497fb2b1a3d496ada6cc5e3a1117ca45203e09b4374f53fee972a

```md
# onto-binary

## MODIFIED Requirements

### Requirement: onto close archives a completed change

`onto close <change>` SHALL archive a completed change. It SHALL run the
framework-install gate, validate the change name, and require ALL of the
following before archiving (each failing case exits non-zero and archives
NOTHING):

- the change's recorded phase is `close` (a change not yet at `close` is
  rejected with guidance to run `onto advance`);
- every dependency listed in the change's `onto-state.yaml` is resolved
  (`DepsResolved` returns empty); otherwise it names the unresolved dependencies;
- the git worktree is clean (a dirty OR undeterminable worktree blocks the
  archive — this is a release-critical operation).

On success it SHALL set `archived: true` in the change's `onto-state.yaml`, then
move `docs/changes/<change>/` to `docs/changes/archive/<YYYY-MM-DD>-<change>/`
(creating the archive directory if needed), and report the archived path. If the
archive target directory already exists it SHALL refuse (non-zero) and move
nothing.

If the archive move itself fails after `archived: true` was written, `onto
close` SHALL roll the `archived` flag back to `false` (re-saving the in-place
`onto-state.yaml`) and exit non-zero, so a failed archive leaves the change
fully un-archived — never marked archived while still at its original path.

#### Scenario: close archives a close-phase change with resolved deps and a clean worktree

- **GIVEN** a change at phase `close` with no unresolved deps in a clean git worktree
- **WHEN** `onto close <change>` runs
- **THEN** `docs/changes/<change>/` is moved to `docs/changes/archive/<date>-<change>/`, its `onto-state.yaml` has `archived: true`, and the command reports the archived path, exiting 0

#### Scenario: a failed archive move leaves the change un-archived

- **GIVEN** a change at phase `close` that passes every archive precondition
- **WHEN** `onto close <change>` runs but the move into the archive directory fails
- **THEN** the command exits non-zero, the change directory remains at its original path, and its `onto-state.yaml` has `archived: false` (the flag was rolled back)

#### Scenario: close refuses a change not at the close phase

- **GIVEN** a change at phase `build`
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero, reports the change is not at `close`, and archives nothing

```
