---
change: framework-ecosystem-model
design-doc: docs/superpowers/specs/2026-07-13-framework-ecosystem-model-design.md
base-ref: e473d63d5e74612a8bd1d5e6e1c0a5a74b9fa4f9
archived-with: 2026-07-13-framework-ecosystem-model
---
# Plan
1. MVP: manifest_schema field + fail-closed guard in catalog.Load (TDD). Commit.
2. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
