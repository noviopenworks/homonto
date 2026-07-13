---
change: local-frameworks
design-doc: docs/superpowers/specs/2026-07-13-local-frameworks-design.md
base-ref: 4ee99ba173ee9b784a44705031c6e7f317bc973c
archived-with: 2026-07-13-local-frameworks
---
# Plan
1. Catalog FS-aware index + mergeFrameworkRoot/LoadWithLocal (TDD). Commit.
2. Config local: acceptance + overlay catalog + expansion (baseDir on Config). Commit.
3. Engine materializeCatalog builds with local overlays; E2E green. Commit.
4. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
