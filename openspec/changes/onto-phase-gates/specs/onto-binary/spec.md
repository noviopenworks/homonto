## ADDED Requirements

### Requirement: Per-phase required artifacts

`internal/ontostate.RequiredArtifacts(phase)` SHALL return the cumulative set of
files that must exist at each workflow phase: `open` → `onto-state.yaml`,
`proposal.md`, `tasks.md`; `design` → those plus `design.md`; `build` → those
plus `plan.md`; `verify` and `close` → those plus `verification.md`. An unknown
phase SHALL return at least the `open` base set. `ValidateSkeleton` SHALL use this
per-phase set so a change's skeleton requirements tighten as it advances.

#### Scenario: build phase requires plan.md

- **GIVEN** a change at phase `build` with `onto-state.yaml`, `proposal.md`, `tasks.md`, `design.md` but no `plan.md`
- **WHEN** `ValidateSkeleton` runs
- **THEN** it returns an error naming `plan.md` as missing

### Requirement: onto advance gates phase transitions

`onto advance <change>` SHALL move a change to the next phase in the fixed order
`open → design → build → verify → close`, and ONLY through that order (no skips,
no reversals). It SHALL run the framework-install gate first. Before advancing it
SHALL verify the transition's precondition:

- the artifacts required to enter the NEXT phase (`RequiredArtifacts(next)`) all
  exist, AND
- when leaving `build`, every `tasks.md` checkbox is checked (at least one
  checkbox present, no unchecked `- [ ]`).

On success it SHALL write the new phase to `onto-state.yaml` and report the
transition. On a failed precondition it SHALL exit non-zero, name what is
missing, and leave the recorded phase unchanged. Advancing a change already at
`close` (or with an unknown phase) SHALL be an error with no write.

#### Scenario: advance open to design when design.md exists

- **GIVEN** a change at phase `open` with `design.md` present (and the open artifacts)
- **WHEN** `onto advance <change>` runs
- **THEN** the recorded phase becomes `design` and the command reports `open → design`, exiting 0

#### Scenario: advance refuses when the next phase's artifact is missing

- **GIVEN** a change at phase `open` with no `design.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero naming `design.md` as missing and the recorded phase stays `open`

#### Scenario: advance out of build requires all tasks checked

- **GIVEN** a change at phase `build` with `plan.md` present but an unchecked `- [ ]` item in `tasks.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating tasks are incomplete and the recorded phase stays `build`

#### Scenario: advance past close is an error

- **GIVEN** a change at phase `close`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating the change is already at the terminal phase and writes nothing

### Requirement: dirty worktree blocks the close transition

`onto advance` SHALL check the workspace's git worktree cleanliness (via `git
status --porcelain`). A dirty worktree SHALL produce a WARNING for a normal
transition (open→design, design→build, build→verify) but SHALL still allow it.
For the release-critical `verify → close` transition a dirty worktree SHALL BLOCK
the advance: the command exits non-zero, reports the dirty worktree, and does not
change the phase.

#### Scenario: dirty worktree warns but allows a normal advance

- **GIVEN** a change at phase `open` (with `design.md`) in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it advances to `design` (exit 0) after printing a dirty-worktree warning

#### Scenario: dirty worktree blocks verify to close

- **GIVEN** a change at phase `verify` (with `verification.md`) in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero reporting the dirty worktree and the recorded phase stays `verify`
