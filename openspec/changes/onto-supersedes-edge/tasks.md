# Tasks — onto-supersedes-edge

## 1. supersedes field + setter + graph edge
- [ ] State.Supersedes []string (ungated); `onto set supersedes <change>
      --change <name>...`; onto graph emits supersedes edges. TDD: setter
      round-trips; graph emits the supersedes edge.

## 2. Verify
- [ ] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet,
      build (incl cmd/onto), `openspec validate --all` green.
