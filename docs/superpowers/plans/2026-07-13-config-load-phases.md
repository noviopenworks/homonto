---
change: config-load-phases
design-doc: docs/superpowers/specs/2026-07-13-config-load-phases-design.md
base-ref: 9eff16820f522cf7e34cf9025e2bdb772a38cfad
---
# Plan
1. Extract decode/migrate/normalize/validate from Load in-order (behavior-preserving,
   config suite is the gate). Commit.
2. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
