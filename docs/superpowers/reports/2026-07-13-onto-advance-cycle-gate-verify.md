# Verification Report — onto-advance-cycle-gate

- **Change:** onto-advance-cycle-gate (F10 — block entering build on a dependency cycle)
- **Date:** 2026-07-13
- **Workflow:** tweak
- **Mode:** light (single capability, one delta spec, ~20-line code diff)
- **Result:** PASS

## Verification checklist (lightweight)

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md tasks checked | PASS (2/2) |
| 2 | Changed files match tasks | PASS — `advance.go` (gate), `advance_cycle_test.go`, `tasks.md` |
| 3 | Build passes (incl. `cmd/onto`) | PASS |
| 4 | Related tests pass | PASS — 150 tests `-race`; the 4 `EnteringBuild*` tests (2 new + 2 existing isolation) pass |
| 5 | No security issues | PASS — read-only graph build over the workspace; the gate only *refuses* (writes nothing on a cycle); no new external input |
| 6 | Code review | Skipped (review_mode=off, tweak) — see below |

## Delta-spec scenario coverage (MODIFIED requirement)

- **entering build refuses a dependency cycle** →
  `TestAdvanceCommand_EnteringBuildRefusesDependencyCycle` (a↔b cycle, isolation
  set → `advance a` errors mentioning "cycle", phase stays `design`).
- **entering build requires isolation** (existing) →
  `TestAdvanceCommand_EnteringBuildBlockedWithoutIsolation` still passes (gate
  ordering preserved: isolation is checked before the cycle check).
- **leaving verify requires a passing result** (existing) → unchanged, still passes.
- Acyclic regression: `TestAdvanceCommand_EnteringBuildAllowedWithAcyclicDeps`
  (a→b, no back-edge) still advances into `build`.

## Build evidence (branch tweak/20260713/onto-advance-cycle-gate)

- `go test -race ./internal/ontostate/... ./internal/ontocli/...` → 150 passed
- `go vet ./...` → clean
- `go build ./...` (incl. `cmd/onto`) → success
- `openspec validate --all` → 16/16 passed

## Code review (review_mode: off)

Skipped per tweak preset. The gate reuses the already-reviewed `buildGraph` +
`detectDepCycles` (from `onto-graph-cycle-check`); it adds only a membership loop
and a refusal in `runAdvance`, ordered after the isolation check. Graph-read
failure fails closed (refuses, does not silently pass). Skip reason in `tasks.md`.

## Branch handling

merge-after-archive: archive runs on the branch first (delta→main sync folds the
cycle clause into the canonical `onto advance gates on phase evidence` requirement
on-branch), then the fully-synced branch merges to main — no transient stale spec.
