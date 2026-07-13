# Verification — onto-stable-change-id (X1 stable id for onto)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (State.ID; NewID crypto/rand 8-hex; onto new sets; immutable; empty legacy) | PASS |
| 3 | Delta scenarios (new assigns stable unique id; legacy loads with empty id, no minting) | PASS |
| 4 | proposal goals (onto identity survives rename; X1 core for the onto plane) | PASS |
| 5 | `go test ./... -race` | PASS |
| 6 | onto suites (ontocli/ontostate) -race | PASS |
| 7 | vet, build (incl cmd/onto), openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual) | PASS |

## Code review (standard, manual)
- `ontostate.NewID` uses crypto/rand → 8 hex; on the (never-observed) rand error returns "" rather than panicking (a change without an id still works, like a legacy state). The no-random rule constrains comet workflow *scripts*, not this Go binary — correct here.
- `onto new` is the ONLY writer of `ID`; `set`/`advance`/`close` Load→mutate→Save, and `Save` round-trips the field, so the id is immutable (proved by TestID_ImmutableAcrossSetAndReload). `Load`/`Parse` never mint an id, so a legacy state loads with empty id and an id never changes meaning across reads.
- `omitempty` yaml/json tags: legacy states stay byte-clean (no `id:` line); the id surfaces in `state --json` and `status` via the State marshal.
- Scoped correctly: onto is homonto's own workflow; the comet/OpenSpec name-matching (external tooling) is untouched — no divergence forced on it.

## Behavior / risk
Low — additive immutable field + generation at creation. Delivers the X1 stable-id core for the onto plane; deps/refs-by-id (the traceability graph) is a documented follow-on. Remaining roadmap: X3 F34 (YAGNI), X2 F42 (de-prioritized), E4 (external infra).
