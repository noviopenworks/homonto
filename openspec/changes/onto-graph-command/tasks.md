# Tasks — onto-graph-command

## 1. onto graph command
- [x] Add `onto graph [--json]`: enumerate active + archived changes → nodes
      (id/change/phase/archived) + depends-on edges from deps; read-only,
      config-independent; deterministic ordering. TDD: nodes for active+archived,
      a depends-on edge, JSON shape.

## 2. Verify
- [x] `go test ./internal/ontocli/... -race`, vet, build (incl cmd/onto),
      `openspec validate --all` green.
