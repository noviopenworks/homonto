# Tasks — typed-plan-operations

## 1. Typed action + validation
- [x] adapter.Action defined type + constants + Valid(); Change.Action is Action.
      ChangeSet.Validate(knownTools) rejects unknown action/tool. Unit tests.

## 2. Engine fail-closed wiring
- [x] engine.Apply validates every set first (before resolve/materialize/write),
      aborting on unknown tool or action. Engine tests: unknown tool aborts,
      unknown action aborts, legal plan applies unchanged.

## 3. Verify
- [x] `go test ./... -race`, vet, build, `openspec validate --all` green.
