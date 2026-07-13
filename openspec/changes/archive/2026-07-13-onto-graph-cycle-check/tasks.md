# Tasks — onto-graph-cycle-check

## 1. Cycle detector + graph wiring + --check
- [x] `detectDepCycles(edges)` over `depends-on` edges (deterministic, ordered
      change-name paths); `onto graph` emits `cycles` in `--json` and a trailing
      `cycles:` section in the human listing; `onto graph --check` exits non-zero
      on a cycle, zero when acyclic. TDD: cycle detected in `--json`; `--check`
      fails on a cycle and passes on an acyclic graph.

## 2. Verify
- [x] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet,
      build (incl `cmd/onto`), `openspec validate --all` green.

<!-- review skipped: review_mode=off (tweak). Cycle detector is a standard
white/gray/black DFS with canonical-rotation dedup; covered by 3 TDD tests
(cycle in --json, --check fails on cycle, --check passes on acyclic). -->
