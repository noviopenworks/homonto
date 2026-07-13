# Tasks — onto-semantic-gates
## 1. onto close semantic evidence gates (workflow-aware)
- [ ] full close requires verify.result==pass + close.merged + guides resolved;
      fix/tweak require verify.result==pass + close.merged (guides not required).
      Clear errors naming the missing evidence. Tests per workflow + per missing token.
## 2. onto advance evidence gates
- [ ] leaving verify requires verify.result==pass; entering build requires
      isolation set. Tests: advance blocked without the evidence, allowed with it.
## 3. Verification
- [ ] `go test ./internal/ontocli/... ./internal/ontostate/... -race`, vet, build,
      `openspec validate --all` green; the N7 conformance suite still passes.
