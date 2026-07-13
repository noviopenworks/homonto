---
change: adapter-registry
design-doc: docs/superpowers/specs/2026-07-13-adapter-registry-design.md
base-ref: 5f252bab64f54b066c2f9d0bed5c7812db3b8e63
archived-with: 2026-07-13-adapter-registry
---
# Plan
1. registry package (Deps/Factory/Registry/Builtins) — TDD. Commit.
2. engine.Build wires via registry.Builtins().Build(deps). Commit.
3. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
