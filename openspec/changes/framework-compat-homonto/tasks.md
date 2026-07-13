# Tasks — framework-compat-homonto

## 1. Catalog Compat field + loose comparator
- [ ] frameworkTOML/[Framework] gain Compat (from [compat].homonto); catalog
      stays version-agnostic. Add satisfiesLoose (strip pre-release/build). Unit
      tests for satisfiesLoose.

## 2. Engine version check + cli wiring
- [ ] engine.Build gains homontoVersion; checks each declared framework's Compat
      fail-closed. cli passes cli.Version (4 sites); test helpers pass a version.
      E2E: [compat].homonto=">=99.0.0" fails; ">=0.1.0" loads.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.
