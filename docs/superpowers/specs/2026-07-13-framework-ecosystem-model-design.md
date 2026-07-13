---
comet_change: framework-ecosystem-model
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-framework-ecosystem-model
status: final
---

# framework-ecosystem-model — Technical Design (E1)

Design-only deliverable for roadmap E1 (F36/F38). OpenSpec is the canonical
spec; the full architecture, phased plan, and decisions live in the change's
`design.md` (`openspec/changes/framework-ecosystem-model/design.md`). This
document is the maintainer-facing summary and the decision record; **no
implementation is part of this change** — it stops at design approval.

## Baseline (already in the code)

The catalog already provides more of E1 than the roadmap implied: transitive
dependency resolution + cycle detection (`internal/catalog/expand.go`), a
duplicate-resource-name conflict error and per-manifest path validation
(`catalog.go`), a versioned catalog (`version.txt`), and builtin-only
enforcement (F35 rejects non-builtin `[frameworks.X]`). The genuine gaps are
manifest schema-versioning, capabilities, compatibility ranges, local/custom
resolution, an explicit conflict policy, and migrations.

## Target model (additive manifest v2)

An optional, backward-compatible superset of today's `framework.toml`:
`manifest_schema` (fail-closed if newer), `[compat].homonto` range,
`[provides].capabilities`, versioned `[dependencies].frameworks` ranges, and
`[dependencies].capabilities`. Every existing builtin manifest stays valid
unchanged. Resolution extends `catalog.Load`/`expandResources` with: manifest-
version guard → compat check → capability resolution → explicit conflict policy →
the existing path/cycle checks.

## Local/custom resolution (the crux)

Reuse the shipped remote-source trust pipeline (`internal/remote/`, digest-
pinned, verify-before-materialize, fail-closed) that subagents already use: a
custom framework is a `local:<path>` (structurally validated) or
`remote:<url>`+digest (trust-verified) catalog overlay loaded through the same
validator as a builtin. This is what lifts F35's blanket rejection *safely*.

## Recommended phasing (MVP first)

1. `manifest_schema` + fail-closed guard, and dependency version-range checks
   (pure additive, builtins unaffected — the smallest verifiable slice).
2. `[compat].homonto` range check.
3. Capabilities + provider index + explicit strict conflict policy.
4. Local/custom resolution via the trust pipeline (needs D1).
5. F38 honest rename; migrations when external manifests land.

## Decisions required before implementation

- **D1** local framework trust: allow `local:` (structural only) and/or
  `remote:`+digest? Trust model for each?
- **D2** capabilities: adopt `name@major` capabilities, or defer and keep
  name-based deps?
- **D3** conflict policy: strict-only (recommended), or also priority/override?
- **D4** F38: real plugin lifecycle, or honest rename now (recommended)?
- **D5** first-implementation scope: MVP only, or through compat + capabilities?

## Risk posture

None in this change (design-only). The purpose is to de-risk implementation by
settling the model and the D1–D5 decisions before any code.
