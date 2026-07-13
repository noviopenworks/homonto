---
comet_change: adapter-registry
role: technical-design
canonical_spec: openspec
status: draft
---

# adapter-registry — Technical Design

Deep design for X3's F33 tool-id-keyed adapter registry. OpenSpec is the
canonical spec; this defers to `openspec/changes/adapter-registry/specs` for
normative requirements; the full API + wiring are in the change's `design.md`.

## Decision

Add `internal/adapter/registry` (Deps, Factory, Registry, `Builtins()`) and have
`engine.Build` construct its adapters via `registry.Builtins().Build(deps)`
instead of a hardcoded slice literal. Adding a built-in adapter becomes one
registration line in `Builtins()`, decoupling the engine's composition root from
each adapter's concrete constructor.

## Why explicit (not global init self-registration)

`Builtins()` returns a fresh registry per call — no global mutable state, no
init() ordering, no import-for-side-effect — so it is hermetic and testable. The
registry package imports the adapters; the adapters do not import the registry,
so there is no cycle.

## Risk posture

Low — a new package plus a single engine wiring site; the same three adapters in
the same order with the same options. Behavior-identity pinned by the engine +
conformance + adapter suites.

## Out of scope

Decoupling the Adapter interface from concrete config/secret/state types (F34
generalization); a distinct ToolID value type (keys are the existing Name()
strings); global self-registration.
