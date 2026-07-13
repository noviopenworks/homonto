## Why
ROADMAP E4 / finding F58 (foundation slice): no performance benchmarks exist, so
there is no way to catch a regression in the hot paths. Add Go benchmarks for two
core operations — config load/expansion and the three-way merge — as the
foundation. Allocation/perf budgets and CI wiring are a follow-up.
## What Changes
- `BenchmarkLoad` (config) and `BenchmarkMerge` (merge), each with `ReportAllocs`,
  runnable via `go test -bench`.
## Impact
- **Code:** `internal/config/config_bench_test.go`, `internal/merge/merge_bench_test.go` (test-only).
- **Spec:** `apply-pipeline` delta (core operations carry benchmark coverage).
- **Out of scope:** allocation budgets, CI benchmark wiring, benchmarks for plan/apply/archive.
