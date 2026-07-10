## Why

onto #1 (foundation), #2 (`onto init`), and #3a (`onto new` + skeleton
validation) are archived. The onto binary can create a change and validate its
skeleton, but it cannot yet ADVANCE a change through the workflow. Per the
dual-binary design the binary enforces structural invariants: "phase transitions
happen only through valid gates" and "dirty worktrees block close/archive". This
change (#3b, the second sub-increment of the onto workflow engine) adds gated
phase advancement.

## What Changes

- Extend `internal/ontostate.RequiredArtifacts(phase)` with per-phase supersets:
  `open` → onto-state.yaml, proposal.md, tasks.md; `design` → + design.md;
  `build` → + plan.md; `verify`/`close` → + verification.md. `ValidateSkeleton`
  (from #3a) automatically tightens as a change advances.
- Add `internal/ontostate.NextPhase(phase) (string, bool)` — the valid successor
  in the fixed order open → design → build → verify → close (returns ok=false at
  `close`, the terminal phase, and for unknown phases).
- Add a build-phase completion check `TasksAllChecked(tasksPath) (bool, error)` —
  true when `tasks.md` has at least one checkbox and no unchecked `- [ ]`.
- Add `onto advance <change>`: the gated phase-transition command. It runs the
  framework-install gate, loads the change's `onto-state.yaml`, derives the
  current phase, and advances to `NextPhase` ONLY IF the transition's
  precondition holds:
  - the artifacts required to ENTER the next phase already exist
    (`ValidateSkeleton`-style check against `RequiredArtifacts(next)`), and
  - leaving `build` additionally requires all `tasks.md` items checked.
  On success it writes the new phase via `ontostate.Save` and reports the
  transition; on a failed precondition it exits non-zero, names what is missing,
  and does NOT change the phase. Advancing past `close` is an error.
- **Dirty-worktree rule:** `onto advance` checks `git status --porcelain` in the
  workspace. A dirty worktree produces a WARNING for normal advances, but BLOCKS
  the release-critical `verify → close` transition (non-zero, no phase change).
- This change does NOT add dependency resolution or archive/close side effects
  beyond the phase write (#3c), nor `onto doctor` (#4) or packaging (#5).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains `onto advance` (valid-gate-only phase transitions with
  per-phase artifact preconditions, build-tasks-complete gate, and dirty-worktree
  blocking for close), plus per-phase `RequiredArtifacts` supersets and
  `NextPhase`/`TasksAllChecked` helpers.

## Impact

- `internal/ontostate`: per-phase `RequiredArtifacts`, `NextPhase`,
  `TasksAllChecked` (+ tests).
- `internal/ontocli`: new `advanceCmd()` (`onto advance`), registered on the
  root; a small git-porcelain helper (os/exec `git status --porcelain`) for the
  dirty-worktree check.
- No new dependency. `onto` stays isolated from homonto's projection pipeline
  (imports none of internal/{cli,engine,config,adapter,catalog}); shelling to
  `git` is allowed (it is the workflow's VCS, not the projection pipeline).
- Advances the onto workflow engine (#3b of 5 onto changes).
