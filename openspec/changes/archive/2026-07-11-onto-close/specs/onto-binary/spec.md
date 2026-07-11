## ADDED Requirements

### Requirement: Dependency resolution

`internal/ontostate.DepsResolved(root string, deps []string) []string` SHALL
return the subset of `deps` that are NOT resolved. A dependency `<dep>` is
resolved iff an archived change directory matching
`docs/changes/archive/*-<dep>` exists under `root`. An empty or nil `deps` SHALL
yield no unresolved dependencies (nil and empty slice are equivalent — both mean
"no dependencies").

#### Scenario: resolved and unresolved dependencies are distinguished

- **GIVEN** a workspace where `docs/changes/archive/2026-07-10-a/` exists but there is no archived `b`
- **WHEN** `DepsResolved(root, ["a","b"])` is called
- **THEN** it returns `["b"]` (a is resolved, b is not)

#### Scenario: no dependencies is always resolved

- **WHEN** `DepsResolved(root, nil)` or `DepsResolved(root, [])` is called
- **THEN** it returns an empty list

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

#### Scenario: close archives a close-phase change with resolved deps and a clean worktree

- **GIVEN** a change at phase `close` with no unresolved deps in a clean git worktree
- **WHEN** `onto close <change>` runs
- **THEN** `docs/changes/<change>/` is moved to `docs/changes/archive/<date>-<change>/`, its `onto-state.yaml` has `archived: true`, and the command reports the archived path, exiting 0

#### Scenario: close refuses a change not at the close phase

- **GIVEN** a change at phase `build`
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero telling the user to `onto advance` to close first, and moves nothing

#### Scenario: close refuses when a dependency is unresolved

- **GIVEN** a close-phase change whose `onto-state.yaml` lists a dep that is not archived
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero naming the unresolved dependency and moves nothing

#### Scenario: close is blocked by a dirty worktree

- **GIVEN** a close-phase change with resolved deps in a workspace with uncommitted changes
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero reporting the dirty worktree and moves nothing

#### Scenario: close refuses to clobber an existing archive entry

- **GIVEN** `docs/changes/archive/<date>-<change>/` already exists
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero and moves nothing
