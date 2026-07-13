# apply-pipeline (delta)

## ADDED Requirements

### Requirement: core operations carry Go benchmarks

`homonto` SHALL carry Go benchmark functions (with allocation reporting) for its
core hot-path operations — config load/expansion and the three-way merge — so a
performance or allocation regression can be measured with `go test -bench`. This is the
regression-tracking foundation; allocation budgets and CI wiring build on it.

#### Scenario: the benchmarks run

- **WHEN** `go test -bench=.` runs in the config and merge packages
- **THEN** the config-load and three-way-merge benchmarks execute and report ns/op and allocations
