# Tasks — catalog-local-overlays

## 1. Overlay merge
- [ ] Refactor Load into mergeSource + LoadOverlays(base, overlays...); Load(fsys)
      = LoadOverlays(fsys) (behavior unchanged). Dependency-range validation runs
      once after all sources merge. Strict conflict falls out of the shared-index
      guard. TDD: no-overlay identity; overlay adds a framework; overlay shadow
      conflict errors; cross-source dependency range validated.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      existing catalog suite unchanged.
