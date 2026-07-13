---
change: typed-plan-operations
design-doc: docs/superpowers/specs/2026-07-13-typed-plan-operations-design.md
base-ref: 4b18e9449ad4c2024af1b1981b9e4a712df2c66e
---
# Plan — typed-plan-operations
1. Typed Action + constants + Valid() + ChangeSet.Validate (TDD, adapter pkg). Commit.
2. engine.Apply validates first, fail-closed (TDD, engine pkg). Commit.
3. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
