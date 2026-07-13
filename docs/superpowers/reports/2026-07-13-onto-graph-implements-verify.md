# Verification — onto-graph-implements (X1 implements edges)

Full verification. PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (Kind; capability nodes; implements edges from specs/*.md) | PASS |
| 3 | Delta scenarios (implements capability; depends-on; read-only) — MODIFIED requirement | PASS |
| 4 | proposal goals (second typed edge; capability traceability) | PASS |
| 5 | E2E: graph --json yields capability node + implements edge; dogfood text output shows depends-on + implements | PASS |
| 6 | `go test ./... -race` + onto suites | PASS |
| 7 | vet, build (incl cmd/onto), openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- graphNode.Kind ("change"|"capability"); change nodes keep id/phase/archived, capability nodes carry the name only. For each change, `os.ReadDir(<dir>/specs)` yields each `<cap>.md` → an implements edge + a deduped capability node. depends-on edges + read-only/config-independence unchanged; deterministic (nodes by kind,name; edges by type,from,to). A change with no specs/ contributes nothing.
- MODIFIED delta keeps the existing canonical requirement header (avoids the MODIFIED-header archive gotcha); synced clean.
- Aside verified: an all-numeric stable id round-trips (yaml.Marshal quotes numeric strings) — the earlier dogfood `no-id` was hand-written unquoted YAML, not a defect.
- Renamed the `cap` local to `capName` (vet: builtin `cap` must be called) — clean.

## Behavior / risk
Low — additive read-only enumerator extension. Second typed edge (implements) of X1's traceability graph. Remaining X1: tests/released-in/supersedes edges (need untracked data — a design decision) + OpenSpec-flow divergence. Remaining roadmap: X3 F34 (YAGNI), X2 F42 (de-prioritized), E4 (external).
