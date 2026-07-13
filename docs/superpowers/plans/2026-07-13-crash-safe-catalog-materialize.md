---
change: crash-safe-catalog-materialize
design-doc: docs/superpowers/specs/2026-07-13-crash-safe-catalog-materialize-design.md
base-ref: 14238456517e86859719aa6f62a6bfe407eba6cb
---
# Plan
1. Stage-then-swap Materialize (TDD: leftover .staging cleaned + content correct). Commit.
2. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
