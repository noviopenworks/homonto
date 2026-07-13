---
comet_change: catalog-local-overlays
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-catalog-local-overlays
status: final
---

# catalog-local-overlays — Technical Design (E1)

OpenSpec is canonical; full approach in the change's `design.md`. The foundation
for local/custom frameworks (E1): a catalog that merges validated overlay
framework sources over the embedded base.

## Decision

Refactor `catalog.Load(fsys)` into `LoadOverlays(base, overlays...)` that merges
each source through the same validation via an extracted `mergeSource`, with the
dependency-range check moved to a single post-merge pass. `Load(fsys)` stays as
`LoadOverlays(fsys)` — base behavior identical. The strict conflict policy (D3)
falls out of the existing shared-index "name mapped to two different paths →
error" guard applied across sources. `version.txt` is read from the base only.

## Why this first

Local-framework support (D1) needs the catalog to read frameworks from the local
filesystem and merge them with the builtins. This is that mechanism, unit-testable
in isolation — the tested building block, exactly as `structproj` shipped before
its adapters consumed it. Config `local:` acceptance + engine materialization
(the consumers) are the next phased changes.

## Risk posture

Low — a structural refactor of Load into mergeSource + a variadic entry point;
base behavior pinned by the existing catalog suite; new tests pin the overlay
merge, strict conflict, and cross-source dependency validation.

## Out of scope

Config `local:<path>` acceptance, engine materialization of local resources,
remote/digest frameworks (later phases).
