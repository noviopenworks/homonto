# Framework [compat].homonto — fail loud on an incompatible homonto version

## Why

Roadmap E1 (F36), the last compatibility mechanism. Now that frameworks can be
shared (local + remote), a framework built for a newer homonto could be
installed on an older one and misbehave silently. A framework should declare the
homonto version range it supports and fail loud at load if the running homonto
is outside it.

## What Changes

- A `framework.toml` MAY declare `[compat].homonto = "<constraint>"` (a version
  constraint over `x.y.z`, reusing the existing comparator; pre-release/build
  metadata on the running version is stripped for the comparison, so a
  `0.1.0-dev` build satisfies `>=0.1.0`).
- The catalog parses it into `Framework.Compat` (staying version-agnostic — it
  does not know the running version).
- The **engine** checks each declared framework's `Compat` against the running
  homonto version (injected into `engine.Build`) and fails closed with a clear
  error before any projection when the version is unsatisfied. A framework with
  no `[compat]` is unconstrained (unchanged).

## Impact

- **Specs:** `framework-expansion` gains a requirement that a framework's
  homonto-compat range is enforced fail-loud.
- **Behavior:** frameworks without `[compat]` are unchanged; new: an incompatible
  framework fails at load with guidance.
- **Risk:** low-per-change but the `engine.Build` version parameter ripples to
  its callers (cli + tests) mechanically.

## Non-goals

- Framework-to-framework compat beyond dependency version ranges (already
  shipped). F38.
