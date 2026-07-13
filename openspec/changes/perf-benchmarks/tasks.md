# Tasks — perf-benchmarks
## 1. Core-op benchmarks
- [x] BenchmarkLoad (config) + BenchmarkMerge (merge) with ReportAllocs; run via go test -bench.
## 2. Verify
- [x] go test -bench=. -benchtime=1x runs the benchmarks; full suite + vet + build + validate green.
