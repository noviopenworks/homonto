---
comet_change: local-frameworks
role: technical-design
canonical_spec: openspec
status: draft
---

# local-frameworks — Technical Design (E1)

OpenSpec is canonical; the full model + wiring is in the change's `design.md`.
This is the flagship E1 capability: local frameworks installed end-to-end
through the same validated path as builtins (D1 = structural validation, no
digest — the user owns their filesystem).

## Decision

`local:<path>` framework = a framework root (framework.toml name==key + resources
at framework-root-relative paths), resolved against the config dir. Wired through:
(1) catalog: FS-aware resource index so Materialize resolves each resource from
its source FS + a single-framework merge (`mergeFrameworkRoot`/`LoadWithLocal`);
(2) config: accept `local:` (F35 kept for other non-builtin), build the catalog
with the config's local overlays (de-globalize `loadedCatalog`, thread baseDir),
expand local frameworks as `builtin:<name>`; (3) engine: build the catalog with
those overlays for materialize. Builtin-only configs are byte-identical.

## Risk posture

Medium — new cross-subsystem behavior. Mitigations: the FS-aware index is
backward-identical for a base-only catalog (full suite green); an end-to-end
acceptance test drives the real path (a local framework's skill materialized by
apply); baseDir is threaded explicitly (no hidden singleton).

## Out of scope

Remote/digest frameworks; `[compat].homonto`; capabilities.
