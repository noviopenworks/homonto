---
change: framework-dependency-ranges
design-doc: docs/superpowers/specs/2026-07-13-framework-dependency-ranges-design.md
base-ref: 986a1406f4f94971c170f2f0b8f2ac45a6a435f4
archived-with: 2026-07-13-framework-dependency-ranges
---
# Plan
1. Comparator (parseVer/satisfies) + dep parsing + Load validation (TDD). Commit.
2. comet manifest ranged deps + out-of-range test. Commit.
3. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
