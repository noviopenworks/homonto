# Framework dependency version ranges, validated fail-loud

## Why

Roadmap E1 (F36), phase-2 of the ecosystem-model design. A framework's
`[dependencies].frameworks` are today bare names with no version constraint, so
a framework that requires a particular version of a dependency cannot say so and
an incompatible pairing would only surface (if ever) at runtime. The E1 exit
gate wants compatibility to "fail loudly." This change adds **dependency version
ranges** validated at catalog load — the first compatibility mechanism, with a
real consumer (the built-in `comet` framework depends on `superpowers`).

Decision taken (per the E1 design's recommended defaults): use a **minimal
hand-rolled `x.y.z` comparator** rather than adding a semver dependency — the
project deliberately minimizes its module graph (pinned toolchains for
govulncheck) and framework versions are simple three-part versions.

## What Changes

- A dependency entry MAY be `"name@<constraint>"` where `<constraint>` is a
  comparator over `x.y.z` (`>=`, `>`, `<=`, `<`, `=`, or a bare exact version).
  A bare `"name"` (today's form) means any version — fully backward-compatible.
- `catalog.Load` validates, after indexing all frameworks, that every ranged
  dependency's target exists and its `version` satisfies the constraint; an
  unsatisfied or unparseable constraint fails loudly, naming the framework, the
  dependency, and the versions.
- The dependency graph used for expansion continues to key on the **name**
  (constraint stripped), so cycle detection and transitive resolution are
  unchanged.
- `comet`'s manifest declares `superpowers@>=0.1.0` and `openspec@>=0.1.0` as the
  first real consumer.

## Impact

- **Specs:** `framework-expansion` gains a requirement that dependency version
  ranges are validated fail-loud at load.
- **Behavior:** bare-name deps behave exactly as today; the only new behavior is
  that a ranged dependency whose target version is out of range fails at load.
- **Risk:** low — additive parsing + a small pure comparator (thoroughly tested)
  + one load-time check; the catalog suite is the safety net.

## Non-goals

- `[compat].homonto` ranges, capabilities, local/custom resolution (later E1
  phases, some gated on D1/D2).
- Pre-release / build-metadata semver (framework versions are plain `x.y.z`).
