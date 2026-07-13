# Tasks — framework-capabilities

## 1. Capability parse + resolution
- [x] frameworkTOML/[Framework] gain provides/required capabilities; parse+
      validate name@major; resolve required capabilities against the provided set
      across all merged sources fail-loud. TDD: unresolved capability errors;
      satisfied/absent load; cross-source (overlay) capability resolves.

## 2. Real consumer
- [x] openspec provides spec-workflow@1; comet requires it. Embedded catalog
      loads; a test proves an unresolved requirement fails loud.

## 3. Verify
- [x] `go test ./... -race`, vet, build, `openspec validate --all` green.
