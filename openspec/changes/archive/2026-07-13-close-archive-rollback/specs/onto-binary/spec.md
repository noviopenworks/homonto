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
