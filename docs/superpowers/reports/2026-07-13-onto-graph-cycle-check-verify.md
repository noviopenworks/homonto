# Verification Report — onto-graph-cycle-check

- **Change:** onto-graph-cycle-check (F10 slice — change-dependency cycle detection)
- **Date:** 2026-07-13
- **Workflow:** tweak
- **Mode:** light (single capability, one delta spec, small real code diff)
- **Result:** PASS

## Verification checklist (lightweight)

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md tasks checked | PASS (2/2) |
| 2 | Changed files match tasks | PASS — `graph.go` (detector + wiring), `cyclecheck_test.go`, `tasks.md` |
| 3 | Build passes (incl. `cmd/onto`) | PASS |
| 4 | Related tests pass | PASS — 148 tests `-race` in ontocli+ontostate |
| 5 | No security issues | PASS — read-only inspection; no new writes, no external input beyond the workspace path already validated by `--dir` |
| 6 | Code review | Skipped (review_mode=off, tweak) — see below |

## Delta-spec scenario coverage (ADDED requirement)

- **graph reports a dependency cycle** → `TestGraph_ReportsCycleInJSON` (a↔b →
  `cycles` array mentions both `a` and `b`).
- **--check fails on a cycle** → `TestGraphCheck_FailsOnCycle` (`Execute` returns
  a non-nil error → non-zero exit).
- **--check passes on an acyclic graph** → `TestGraphCheck_PassesOnAcyclic`.

## Dogfood evidence (`onto graph` on a real a↔b workspace)

```
a (no-id, build)
  → depends-on b
b (no-id, build)
  → depends-on a
cycles:
  a → b → a
```
Plain `graph` exits 0; `graph --check` prints the same and exits non-zero (1),
with no cobra usage dump (`SilenceUsage`/`SilenceErrors` set on root).

## Build evidence (branch tweak/20260713/onto-graph-cycle-check)

- `go test -race ./internal/ontostate/... ./internal/ontocli/...` → 148 passed
- `go vet ./...` → clean
- `go build ./...` (incl. `cmd/onto`) → success
- `openspec validate --all` → 16/16 passed

## Code review (review_mode: off)

Skipped per tweak preset. The detector is a textbook white/gray/black DFS with
canonical-rotation dedup for determinism; the `cycles` JSON field is additive
(existing `graphJSON` tests ignore it and still pass); `--check` only changes the
exit code, never mutates state. Skip reason recorded in `tasks.md`.

## Branch handling

merge-after-archive: archive runs on the branch first (delta→main sync ADDs the
cycle-detection requirement to the canonical `onto-binary` spec on-branch), then
the fully-synced branch merges to main — main never carries a transient
stale-spec commit.
