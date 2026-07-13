# Tasks — onto-deviates-from-edge

## 1. deviates-from field + setter + graph edge
- [x] `State.DeviatesFrom []string` (ungated); `onto set deviates-from <change>
      --from <name>...`; `onto graph` emits `deviates-from` edges. TDD: setter
      round-trips leaving other fields unchanged; graph emits the edge.

## 2. Verify
- [x] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet,
      build (incl `cmd/onto`), `openspec validate --all` green.

<!-- review skipped: review_mode=off (tweak). Mechanical mirror of
onto-supersedes-edge (reviewed in cd76c07); identical setter/edge pattern. -->
