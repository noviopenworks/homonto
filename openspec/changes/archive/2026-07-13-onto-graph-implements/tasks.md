# Tasks — onto-graph-implements

## 1. Capability nodes + implements edges
- [x] onto graph emits capability nodes (kind) and implements edges (change ->
      capability from specs/<cap>.md); change nodes get kind:"change". Read-only,
      deterministic. TDD: a change with a specs/<cap>.md yields the capability
      node + implements edge; JSON shape carries kind.

## 2. Verify
- [x] `go test ./internal/ontocli/... -race`, vet, build (incl cmd/onto),
      `openspec validate --all` green.
