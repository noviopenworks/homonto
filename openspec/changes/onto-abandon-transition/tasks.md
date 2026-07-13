# Tasks — onto-abandon-transition

## 1. Abandoned field + onto abandon + advance-refusal + graph marker
- [ ] `State.Abandoned bool` (ungated); `onto abandon <change>` (idempotent, gate
      + valid-name + loadable, refuses if archived); `onto advance` refuses an
      abandoned change; `onto graph` marks abandoned (`--json` field + human
      suffix). TDD: abandon sets the flag leaving phase unchanged; advance refuses
      an abandoned change; abandon refuses an archived change; graph marks it.

## 2. Verify
- [ ] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet,
      build (incl `cmd/onto`), `openspec validate --all` green.
