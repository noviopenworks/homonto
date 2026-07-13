---
comet_change: framework-compat-homonto
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-framework-compat-homonto
status: final
---

# framework-compat-homonto — Technical Design (E1)

OpenSpec is canonical; full approach in the change's `design.md`. `[compat].
homonto` is enforced fail-loud: the catalog stays version-agnostic (stores
`Framework.Compat`), the engine (given the running version via `Build`) checks
each declared framework's constraint before projection. Pre-release/build suffix
on the running version is stripped (`satisfiesLoose`) so a `0.1.0-dev` build
satisfies `>=0.1.0`. Meaningful now that frameworks can be shared (local/remote).

## Risk posture

Low logic (reuses the comparator); the `engine.Build` `homontoVersion` parameter
ripples mechanically to cli (4 sites) + test helpers, compiler-checked.

## Out of scope

A leaf buildinfo package (avoids changing release ldflags -X targets); F38.
