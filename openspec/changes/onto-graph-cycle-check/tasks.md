# Tasks — onto-graph-cycle-check

## 1. Cycle detector + graph wiring + --check
- [ ] `detectDepCycles(edges)` over `depends-on` edges (deterministic, ordered
      change-name paths); `onto graph` emits `cycles` in `--json` and a trailing
      `cycles:` section in the human listing; `onto graph --check` exits non-zero
      on a cycle, zero when acyclic. TDD: cycle detected in `--json`; `--check`
      fails on a cycle and passes on an acyclic graph.

## 2. Verify
- [ ] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet,
      build (incl `cmd/onto`), `openspec validate --all` green.
