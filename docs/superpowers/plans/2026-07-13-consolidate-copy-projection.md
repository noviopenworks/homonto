---
change: consolidate-copy-projection
design-doc: docs/superpowers/specs/2026-07-13-consolidate-copy-projection-design.md
base-ref: b83efecedce97958295ca34fd3585d0f8e26dcf4
archived-with: 2026-07-13-consolidate-copy-projection
---
# Plan — consolidate copy-mode projection
Safety net: conformance + per-adapter copy-mode tests green each step.
1. copyproj core (TDD): Name/Plan/Apply + recordedCopyHashes + keyPrefix. Commit.
2. claude migration: route copy-mode through copyproj. Commit.
3. opencode migration: same. Commit.
4. verify: go test ./... -race, vet, build, openspec validate --all. Commit.
