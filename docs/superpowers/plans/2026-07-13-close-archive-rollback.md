---
change: close-archive-rollback
design-doc: docs/superpowers/specs/2026-07-13-close-archive-rollback-design.md
base-ref: fae08c9e0d499be9b5643d0ef957a1a243837bd9
---
# Plan
1. runClose rolls back archived on move failure (TDD: injected failure). Commit.
2. Verify: go test ./internal/ontocli/... -race, vet, build, openspec validate. Commit.
