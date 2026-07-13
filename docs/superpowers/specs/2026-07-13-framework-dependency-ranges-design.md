---
comet_change: framework-dependency-ranges
role: technical-design
canonical_spec: openspec
status: draft
---

# framework-dependency-ranges — Technical Design (E1 phase-2)

OpenSpec is the canonical spec; the full approach is in the change's
`design.md`. This is the E1 phase-2 slice of the ecosystem-model design:
dependency version ranges, the first compatibility mechanism with a real
consumer.

## Decision

Add `"name@<constraint>"` dependency version ranges validated fail-loud at
`catalog.Load`, using a **minimal hand-rolled `x.y.z` comparator** — not a semver
dependency — because framework versions are plain three-part versions and the
project deliberately minimizes its module graph (pinned toolchains for
govulncheck). Bare names remain "any version"; the expansion graph keys on the
name so cycle/transitive resolution is unchanged. `comet` declares
`superpowers@>=0.1.0` + `openspec@>=0.1.0` as the first real consumer.

## Risk posture

Low — additive parsing + a small pure comparator (thoroughly unit-tested) + one
load-time validation pass. Bare-name deps behave exactly as today; the catalog +
expand suites are the safety net.

## Out of scope

`[compat].homonto`, capabilities, local/custom resolution (later E1 phases, some
gated on D1/D2); pre-release/build-metadata semver.
