# Tasks — onto-advance-cycle-gate

## 1. entering-build cycle gate
- [x] `runAdvance` refuses entering `build` when the change is in a `depends-on`
      cycle (reuse `buildGraph` + `detectDepCycles`), naming the cycle and writing
      nothing; isolation gate and all other transitions unchanged. TDD: a change
      in an a↔b cycle cannot advance design→build (phase unchanged); an acyclic
      change still advances.

## 2. Verify
- [x] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet,
      build (incl `cmd/onto`), `openspec validate --all` green.

<!-- review skipped: review_mode=off (tweak). Gate reuses the reviewed
detectDepCycles/buildGraph; only adds a membership check + refusal in
runAdvance. Covered by 2 TDD tests (cycle refused, acyclic advances). -->
