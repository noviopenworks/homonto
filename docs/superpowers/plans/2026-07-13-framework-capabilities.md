---
change: framework-capabilities
design-doc: docs/superpowers/specs/2026-07-13-framework-capabilities-design.md
base-ref: ca7e40cc5dcca70e30e9c3b94cfaf5c9253000ad
archived-with: 2026-07-13-framework-capabilities
---
# Plan
1. Capability parse + resolution in catalog (TDD) + builtin consumer. Commit.
2. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
