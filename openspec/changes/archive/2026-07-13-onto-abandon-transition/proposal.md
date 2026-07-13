# Proposal — onto-abandon-transition

## Why

onto's workflow has one terminal state — `close` (a change that completed and
archived). But real changes are also *cancelled*: superseded, deprioritized, or
abandoned mid-flight. Today onto has no way to record that. A cancelled change
sits at whatever phase it stalled in, forever counted as active by `onto status`
and `onto graph`, and `onto advance` will happily keep pushing it forward. This
is the N1-residual "abandon transition" gap: the state machine lacks an
unsuccessful terminal.

## What

- `State.Abandoned bool` (ungated — mirrors `Archived`).
- `onto abandon <change>` — marks a change abandoned (idempotent). Refuses to
  abandon a change that already `Archived` (a completed change is not abandonable).
- `onto advance` refuses to advance an abandoned change (it is terminal), leaving
  the phase unchanged.
- `onto graph` marks abandoned changes (`abandoned: true` in `--json`, an
  `abandoned` suffix in the human listing) so a cancelled change is never shown as
  ordinary active work.

## Scope

- **In:** the field, the `onto abandon` command, the advance-refusal, the graph
  marker, TDD tests, delta spec.
- **Out (non-goals):** un-abandon / reopen (a separate decision); archiving or
  moving an abandoned change's directory (abandon only marks state — housekeeping
  is the operator's call); changing `onto close` or the close evidence gates.
