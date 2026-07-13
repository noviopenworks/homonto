# Verification — onto-graph-command (X1 traceability graph)

Full verification. PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (read-only enumerator; nodes + depends-on edges; text/--json) | PASS |
| 3 | Delta scenarios (active+archived nodes with a depends-on edge; read-only, no config) | PASS |
| 4 | proposal goals (surfaces change dependency relationships) | PASS |
| 5 | E2E-ish: graph --json over an active change depending on an archived one | PASS |
| 6 | `go test ./... -race` + onto suites | PASS |
| 7 | vet, build (incl cmd/onto), openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- Read-only + config-independent (mirrors `onto status`): enumerates `docs/changes/*` (skip `archive`) and `docs/changes/archive/*`, `ontostate.Classify` per dir, never writes, never needs homonto.toml (test `TestGraphCommand_ReadOnlyNoConfig`). A malformed/missing-state change still yields a node labeled by directory (F14 — no silent drop).
- Nodes carry the stable id + name + phase + archived; edges are `depends-on` (from change → each deps entry). Deterministic ordering (nodes by change, edges by from,to). `--json` emits `{nodes, edges}`; text is an adjacency listing.
- Scoped: `depends-on` over changes only; the richer typed edges (implements/tests/supersedes/deviates-from/released-in) and edges-by-id are explicit follow-ons.

## Behavior / risk
Low — additive read-only command. First slice of X1's traceability graph, built on the stable-id core. Remaining X1: richer typed edges + CI validation + the OpenSpec-flow divergence decision. Remaining roadmap: X3 F34 (YAGNI), X2 F42 (de-prioritized), E4 (external infra).
