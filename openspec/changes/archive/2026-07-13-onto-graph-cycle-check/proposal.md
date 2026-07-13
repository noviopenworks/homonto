# Proposal — onto-graph-cycle-check

## Why

onto records change→change dependencies (`deps`, surfaced as `depends-on` edges
by `onto graph`), but nothing detects a **dependency cycle** among changes. A
cycle (A depends on B depends on A) is a structural error — no valid build order
exists — yet onto today reports it as ordinary edges and moves on. F10 (N2) calls
for a dep resolver with cycle detection; X1's exit gate calls for "typed edges
validated in CI." This delivers the cycle-detection slice: a structural check onto
can enforce under B1 (a cycle is a fact about the recorded data, not a judgment).

The catalog already detects *framework* dependency cycles (`internal/catalog/
expand.go`); this is the distinct *change*-dependency graph, which had no check.

## What

- A deterministic cycle detector over the `depends-on` edges of the change graph
  (`internal/ontocli`), reporting each cycle as an ordered change-name path.
- `onto graph` surfaces detected cycles: in `--json` as a `cycles` array; in the
  human listing as a trailing `cycles:` section. Absence of cycles changes nothing.
- `onto graph --check`: exits non-zero (with the cycle(s) reported) when any
  dependency cycle exists, and zero otherwise — a CI-usable structural gate.

## Scope

- **In:** the cycle detector, its wiring into `onto graph` output, the `--check`
  exit-code gate, TDD tests, delta spec.
- **Out (non-goals):** blocking `onto advance`/`onto new` on a cycle (the fuller
  F10 "blocks entering build" slice — a gate-logic change, separable); resolving
  `deps` against a registry / date-anchored matching; cycles among any edge type
  other than `depends-on`.
