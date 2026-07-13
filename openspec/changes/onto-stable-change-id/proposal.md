# onto: a stable change ID, independent of the change name

## Why

Roadmap X1 (stable IDs). X1's original tension — stable IDs vs OpenSpec's
name-matching — is about the external comet/OpenSpec tooling. But **onto is
homonto's own binary-enforced workflow**, keyed today entirely on the change name
/ directory (`docs/changes/<name>/`). A rename loses all identity, and a
dependency (`deps: [<name>]`) or any cross-reference is a fragile string match —
the exact fragility X1 names. onto can carry a stable identity without touching
OpenSpec: a content-independent ID assigned once at creation.

## What Changes

- `onto-state.yaml` gains a stable `id` — a short random hex identifier assigned
  once by `onto new` and never rewritten. It survives a change rename and is the
  durable anchor for future traceability (deps-by-id, cross-refs).
- `onto new` generates the id (crypto/rand); `onto set`/`advance`/`close`
  preserve it verbatim (immutable); `onto state --json` and `onto status` surface
  it.
- On-read of a legacy state with no `id`, the value stays empty (backward
  compatible) — the id is assigned only at creation, never retroactively minted
  (so it cannot change meaning across reads).

## Impact

- **Specs:** `onto-binary` gains a requirement that a change carries a stable,
  name-independent id assigned at creation and never mutated.
- **Behavior:** additive; every existing command works, legacy states load with
  an empty id.
- **Risk:** low — an additive immutable field + generation at `onto new`; Go
  tests pin generation, uniqueness, and immutability.

## Non-goals

- Migrating `deps` or references from names to ids (the traceability-graph
  follow-on); retroactively minting ids for legacy changes; any change to the
  comet/OpenSpec flow (which keeps its name-matching).
