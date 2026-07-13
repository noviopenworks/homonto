---
change: catalog-local-overlays
design-doc: docs/superpowers/specs/2026-07-13-catalog-local-overlays-design.md
base-ref: f2587f23d3e234d291744a749798213e6f503d63
---
# Plan
1. Load -> mergeSource + LoadOverlays(base, overlays...) (TDD: overlay add/shadow/dep). Commit.
2. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
