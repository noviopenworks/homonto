# Proposal — onto-deviates-from-edge

## Why

X1's traceability graph now carries `depends-on`, `implements`, and
`supersedes` edges. The last change→target relationship edge onto can track
without inventing a concept it lacks is **`deviates-from`**: an explicit,
honest record of where a change's implementation diverges from a decision,
spec, or prior change. The verify flow already reasons about "Implementation
Divergence" (spec drift); recording it as a first-class, queryable edge lets
the graph answer "what deviated from what" — a stated X1 goal.

`tests` and `released-in` are deliberately out of scope: they need concepts
onto does not model (a test registry, a release model) and stay design-gated.

## What

- `State.DeviatesFrom []string` (ungated — never blocks a phase transition),
  mirroring `Deps`/`Supersedes`.
- `onto set deviates-from <change> --from <name> [--from <name> ...]` — sets
  the list; repeatable `--from` flag keeps names with edge characters
  unambiguous. Mirrors the `deps`/`supersedes` setters exactly.
- `onto graph` emits a `deviates-from` edge (change → each named target),
  deterministic and read-only, alongside the existing edge types.

## Scope

- **In:** the field, the setter, the graph edge, TDD tests, delta spec.
- **Out (non-goals):** `tests`/`released-in` edges (design-gated); CI edge
  validation and the OpenSpec-divergence question (maintainer decisions);
  any semantic gate that consumes the edge; homonto-engine work.
