# Verification Report — onto-deviates-from-edge

- **Change:** onto-deviates-from-edge (X1 traceability — deviates-from edge)
- **Date:** 2026-07-13
- **Workflow:** tweak
- **Mode:** full (scale bumped by changed-file count; the real code diff is 5 files —
  `state.go`, `set.go`, `graph.go`, `deviatesfrom_test.go`, `tasks.md` — the rest
  is `.comet/` tracking noise)
- **Result:** PASS

## Verification checklist

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md tasks checked | PASS (2/2) |
| 2 | Implementation matches proposal decisions | PASS — deviates-from mirrors supersedes/deps (list-of-names, ungated, `onto set deviates-from --from`, graph edge) |
| 3 | Design Doc consistency | N/A — tweak has no Superpowers Design Doc; the delta spec `onto-binary/spec.md` is the design authority and matches the implementation |
| 4 | All delta-spec scenarios pass | PASS — see coverage below |
| 5 | proposal.md goals satisfied | PASS — deviates-from relationship + graph edge delivered |
| 6 | No delta/spec contradictions | PASS |

## Delta-spec scenario coverage

- **graph lists dependency, implements, supersedes, and deviates-from edges** →
  `TestGraph_DeviatesFromEdge` (asserts `{from:alpha,to:adr-7,type:"deviates-from"}`);
  existing tests cover the other three edge types.
- **onto set deviates-from records the relationship** →
  `TestSetDeviatesFrom_RoundTripsLeavingOthers` (asserts `DeviatesFrom == [adr-1 adr-2]`
  and `ID` unchanged — "leaving other fields unchanged").
- **graph is read-only and needs no config** → existing graph tests run without `homonto.toml`.

## Build evidence (branch tweak/20260713/onto-deviates-from-edge)

- `go test -race ./internal/ontostate/... ./internal/ontocli/...` → 145 passed
- `go vet ./...` → clean
- `go build ./...` (incl. `cmd/onto`) → success
- `openspec validate --all` → 16/16 passed

## Code review (review_mode: off)

Skipped per tweak preset. This is a mechanical mirror of `onto-supersedes-edge`
(reviewed at `cd76c07`, no CRITICAL/IMPORTANT): identical `StringArrayVar` +
`runTransition` setter and identical graph-edge emission. Same accepted MINOR
characteristics (set-replaces-on-empty, no dedup/self-loop guard). Skip reason
recorded in `tasks.md`.

## Branch handling

merge-after-archive: archive runs on the branch first (delta→main sync folds the
deviates-from edge into the canonical `onto-binary` spec on-branch), then the
fully-synced branch merges to main — main never carries a transient stale-spec commit.
