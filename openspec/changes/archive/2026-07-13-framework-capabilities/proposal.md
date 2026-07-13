# Framework capabilities: depend on an interface, not a framework name

## Why

Roadmap E1 (F36), the capability model (design decision D2: `name@major`
capability strings). Now that frameworks can be shared (local + remote), a
framework should be able to depend on a *capability* (e.g. "spec-workflow@1")
that any provider satisfies, rather than hard-coding a specific framework name —
looser coupling for the ecosystem. A framework declares the capabilities it
provides and the capabilities it requires; the catalog resolves required
capabilities to providers at load, failing loud on an unresolved requirement.

## What Changes

- A `framework.toml` MAY declare `[provides].capabilities = ["name@major", …]`
  and `[dependencies].capabilities = ["name@major", …]`. A capability is a
  `name@major` string (name plus a non-negative integer major version).
- `catalog.Load`/`LoadOverlays`/`LoadWithLocal` validate the capability format
  and, after indexing all frameworks (base + overlays), resolve every required
  capability against the set provided across all frameworks — an unresolved
  requirement fails loud, naming the framework and the capability. Multiple
  providers of one capability are allowed (it is an interface, not a resource).
- Consumer: `openspec` provides `spec-workflow@1`; `comet` requires
  `spec-workflow@1` (it already depends on openspec by name — capabilities make
  the relationship interface-based).

## Impact

- **Specs:** `framework-expansion` gains a requirement that capability
  requirements are resolved fail-loud at load.
- **Behavior:** frameworks without capability declarations are unchanged; the
  new behavior is that a required capability with no provider fails at load.
- **Risk:** low — additive parsing + a load-time resolution pass; the catalog
  suite pins existing behavior.

## Non-goals

- Selecting *which* provider satisfies a capability when several do (any
  provider suffices; resource-name conflicts remain the strict-error case).
- `[compat].homonto` (needs version injection). Capability major-range
  requirements beyond exact `name@major` match.
