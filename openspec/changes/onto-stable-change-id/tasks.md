# Tasks — onto-stable-change-id

## 1. Stable id field + generation + immutability
- [ ] State gains `id` (yaml/json); onto new generates a short random hex id;
      set/advance/close preserve it; state --json / status surface it; legacy
      (no id) loads empty, never retro-minted. TDD: new produces a well-formed
      unique id; a second change differs; the id survives advance/set unchanged.

## 2. Verify
- [ ] `go test ./internal/ontostate/... ./internal/ontocli/... -race`, vet,
      build (incl. cmd/onto), `openspec validate --all` green.
