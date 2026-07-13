# Tasks — crash-safe-catalog-materialize

## 1. Stage-then-swap skill materialization
- [ ] catalog.Materialize writes each skill into a staging dir, then atomically
      swaps it into place (remove leftover staging first; RemoveAll(dst)+Rename
      on success). TDD: a walk that fails mid-skill leaves the prior dst intact
      and no partial dst; success writes identical bytes.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      existing catalog + engine materialize suites pass unchanged.
