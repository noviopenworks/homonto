# Tasks — framework-dependency-ranges

## 1. Comparator + dep-range validation
- [x] Add a pure x.y.z comparator (parseVer/satisfies) + unit tests. Parse
      "name@constraint" deps (bare name = any), carry constraints, and validate
      at catalog.Load fail-loud (unknown dep / out-of-range / unparseable).
      Cycle/transitive expansion unchanged (keys on name).

## 2. Real consumer
- [x] comet manifest declares superpowers@>=0.1.0, openspec@>=0.1.0. Catalog
      loads (both at 0.1.0). Tests prove out-of-range fails loud.

## 3. Verify
- [x] `go test ./... -race`, vet, build, `openspec validate --all` green.
