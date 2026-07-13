# onto-binary

## MODIFIED Requirements

### Requirement: onto advance gates on phase evidence

`onto advance` SHALL require the evidence a phase must have before leaving or
entering it, beyond artifact existence and checked tasks. Leaving `verify` SHALL
require `verify.result == pass` (never `pending` or `fail`). Entering `build` SHALL
require `isolation` chosen (`branch` or `worktree`), so planning work is never
committed unisolated. Entering `build` SHALL also require that the change is not
part of a change-dependency (`depends-on`) cycle: if the change graph contains a
cycle through this change, no valid build order exists, so `onto advance` MUST
refuse and name the cycle. A missing token or a dependency cycle SHALL block the
transition with a clear error, writing nothing.

#### Scenario: leaving verify requires a passing result

- **GIVEN** a change at `verify` whose `verify.result` is `pending`
- **WHEN** `onto advance` runs
- **THEN** it refuses naming the missing passing verification and leaves the phase unchanged

#### Scenario: entering build requires isolation

- **GIVEN** a change ready to advance from `design` to `build` with no `isolation` set
- **WHEN** `onto advance` runs
- **THEN** it refuses naming the missing isolation and leaves the phase at `design`

#### Scenario: entering build refuses a dependency cycle

- **GIVEN** a change at `design` with `isolation` set, whose `deps` form a cycle
  with another change that depends back on it
- **WHEN** `onto advance` runs
- **THEN** it refuses naming the dependency cycle and leaves the phase at `design`
