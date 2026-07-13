# onto-binary (delta)

## ADDED Requirements

### Requirement: onto close requires close-phase evidence

`onto close` SHALL, in addition to its existing phase, dependency, and
clean-worktree gates, require the close-phase evidence tokens that the workflow
produces, and SHALL refuse to archive (with a clear error naming the missing
evidence) when they are absent. For a `full` workflow it SHALL require
`verify.result == pass`, `close.merged == true`, and `guides` resolved (`updated`
or a `waived:<reason>`, never `pending` or empty). For a `fix` or `tweak` preset it
SHALL require the reduced set those presets produce — `verify.result == pass` and
`close.merged == true` — and SHALL NOT require `guides`. This makes archiving gate
on real evidence (B1: the token is present and well-formed), not merely on file
existence and checked boxes.

#### Scenario: full close is refused without a passing verification

- **GIVEN** a `full` change at the close phase whose `verify.result` is `pending` (or `fail`)
- **WHEN** `onto close` runs
- **THEN** it refuses with an error naming the missing passing verification, and archives nothing

#### Scenario: full close is refused without resolved guides

- **GIVEN** a `full` change at close with `verify.result == pass` and `close.merged == true` but `guides` still `pending`
- **WHEN** `onto close` runs
- **THEN** it refuses naming the unresolved guides, and archives nothing

#### Scenario: a tweak close does not require guides

- **GIVEN** a `tweak` change at close with `verify.result == pass` and `close.merged == true` and `guides` unset
- **WHEN** `onto close` runs
- **THEN** the reduced preset gate is satisfied and (with deps + clean worktree) the change archives

### Requirement: onto advance gates on phase evidence

`onto advance` SHALL require the evidence a phase must have before leaving or
entering it, beyond artifact existence and checked tasks. Leaving `verify` SHALL
require `verify.result == pass` (never `pending` or `fail`). Entering `build` SHALL
require `isolation` chosen (`branch` or `worktree`), so planning work is never
committed unisolated. A missing token SHALL block the transition with a clear
error, writing nothing.

#### Scenario: leaving verify requires a passing result

- **GIVEN** a change at `verify` whose `verify.result` is `pending`
- **WHEN** `onto advance` runs
- **THEN** it refuses naming the missing passing verification and leaves the phase unchanged

#### Scenario: entering build requires isolation

- **GIVEN** a change ready to advance from `design` to `build` with no `isolation` set
- **WHEN** `onto advance` runs
- **THEN** it refuses naming the missing isolation and leaves the phase at `design`
