# Proposal — onto-advance-cycle-gate

## Why

`onto-graph-cycle-check` gave onto the ability to *detect* change-dependency
cycles (`detectDepCycles`), but detection is opt-in (`onto graph --check`).
Nothing stops a change that is part of a dependency cycle from advancing into
`build` and being worked on — even though a cycle means no valid build order
exists. This is the F10 "dep resolver blocks entering build" intent: a structural
precondition onto can enforce under B1 (a cycle is a fact about the recorded
`deps`, not a judgment).

## What

Extend the `onto advance` entering-`build` gate: in addition to requiring
`isolation`, it refuses to enter `build` when the change participates in a
`depends-on` cycle, naming the cycle and writing nothing. Reuses the existing
`buildGraph` + `detectDepCycles` from `onto graph` — no new detection logic.

## Scope

- **In:** the entering-build cycle gate in `runAdvance`, TDD tests, delta spec.
- **Out (non-goals):** gating any transition other than entering `build`;
  date-anchored dep resolution against a registry (the remaining, larger F10
  slice); changing detection semantics (cycles are still the `depends-on`
  subgraph only).
