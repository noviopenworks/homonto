---
comet_change: framework-capabilities
role: technical-design
canonical_spec: openspec
status: draft
---

# framework-capabilities — Technical Design (E1)

OpenSpec is canonical; full approach in the change's `design.md`. The D2
capability model: a framework declares `[provides].capabilities` and
`[dependencies].capabilities` as `name@major`; the catalog resolves required
capabilities against providers (across base + overlays) fail-loud at load.
Consumer: openspec provides `spec-workflow@1`, comet requires it. Now meaningful
because frameworks can be shared (local/remote), so depending on an interface
rather than a framework name is real loose coupling.

## Risk posture

Low — additive parse + a load-time resolution pass; frameworks without
capabilities are unchanged (catalog + expand suites green). Exact `name@major`
match (a major bump is an incompatible interface change); multiple providers
allowed.

## Out of scope

Provider selection when several satisfy; capability major *ranges*;
`[compat].homonto`.
