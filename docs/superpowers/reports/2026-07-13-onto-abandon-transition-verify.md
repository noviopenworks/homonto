# Verification Report — onto-abandon-transition

- **Change:** onto-abandon-transition (N1 residual — unsuccessful terminal state)
- **Date:** 2026-07-13
- **Workflow:** tweak
- **Mode:** light (single capability, one delta spec)
- **Result:** PASS

## Verification checklist (lightweight)

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md tasks checked | PASS (2/2) |
| 2 | Changed files match tasks | PASS — `state.go` (field), `abandon.go` (command), `root.go` (register), `advance.go` (refusal), `graph.go` (marker), `abandon_test.go` |
| 3 | Build passes (incl. `cmd/onto`) | PASS |
| 4 | Related tests pass | PASS — 154 tests `-race` |
| 5 | No security issues | PASS — abandon only sets a flag on an existing loadable change under the framework gate + valid-name checks; refuses an archived change; no external input, no deletion/move |
| 6 | Code review | Skipped (review_mode=off, tweak) — see below |

## Delta-spec scenario coverage (ADDED requirement)

- **abandon marks the change** → `TestAbandon_MarksChangeLeavingPhase`
  (`Abandoned == true`, phase unchanged).
- **advance refuses an abandoned change** → `TestAdvance_RefusesAbandonedChange`
  (error mentions "abandon", phase unchanged).
- **abandon refuses an archived change** → `TestAbandon_RefusesArchivedChange`.
- Graph marker → `TestGraph_MarksAbandoned` (`abandoned: true` in `--json`).

## Build evidence (branch tweak/20260713/onto-abandon-transition)

- `go test -race ./internal/ontostate/... ./internal/ontocli/...` → 154 passed
- `go vet ./...` → clean
- `go build ./...` (incl. `cmd/onto`) → success
- `openspec validate --all` → 16/16 passed

## Code review (review_mode: off)

Skipped per tweak preset. `onto abandon` mirrors `onto close`'s command shape
(gate → valid-name → load → guarded write); `Abandoned` mirrors the existing
`Archived bool`; the advance-refusal and graph marker are additive one-liners.
Idempotent; refuses an archived change; writes nothing on refusal. Skip reason in
`tasks.md`.

## Branch handling

merge-after-archive: archive runs on the branch first (delta→main sync ADDs the
abandon requirement to the canonical `onto-binary` spec on-branch), then the
fully-synced branch merges to main — no transient stale spec.
