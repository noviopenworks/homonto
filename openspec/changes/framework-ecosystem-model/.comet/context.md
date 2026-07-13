# Comet Design Handoff

- Change: framework-ecosystem-model
- Phase: design
- Mode: compact
- Context hash: 9b866b2a1490e1e4a625b80b3faf3d49f5d977375314c79fcd424167a709fd65

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/framework-ecosystem-model/proposal.md

- Source: openspec/changes/framework-ecosystem-model/proposal.md
- Lines: 1-52
- SHA256: 2a44c62fc0181d912a9c569331438aad9be9031d2cb23d0e37d58d3239d2c9e6

```md
# A real framework ecosystem model (design)

## Why

Roadmap E1 (F36/F38). Today a framework is an embedded `framework.toml`
(name, content version, unversioned `[dependencies].frameworks`, and
skill/command/subagent name→path maps). The catalog already resolves
dependencies transitively with cycle detection and rejects a resource name
mapped to two different paths, and `F35` rejects any non-builtin
`[frameworks.X]` source at load. What is missing, per the E1 exit gate, is the
model that lets **a fourth framework or a local framework install through the
same validated, versioned path**:

- a **manifest schema version** (distinct from a framework's content version) so
  an external manifest from a newer schema fails closed instead of being
  silently half-read;
- **provided/required capabilities** so a framework can depend on a *capability*
  (e.g. "spec-workflow") rather than a hard framework name;
- **compatibility ranges** (against the homonto version and against dependency
  versions) so an incompatible pairing fails loudly, not at runtime;
- **local/custom framework resolution** — the validated path that lifts F35's
  blanket rejection for a trusted local source;
- an **explicit conflict policy** for two frameworks providing the same resource
  or capability (today: different-path = hard error, same = first-wins collapse);
- **manifest migrations** across schema versions;
- **F38**: either a real plugin *lifecycle* capability or an honest rename of the
  current enable/disable `[plugins]` projection.

## What Changes

This change delivers the E1 ecosystem-model **design** (the manifest schema,
resolution/validation pipeline, capability + compatibility model, local-source
trust reuse, conflict policy, phased plan, and the D1-D5 maintainer decisions)
**and implements the phase-1 MVP** — the additive `manifest_schema` field plus a
fail-closed guard in the framework loader — which the design shows is independent
of D1-D4. The broader model (capabilities, compat ranges, local/custom
resolution, F38) remains design for phased follow-on changes pending those
decisions.

## Impact

- **Specs:** the eventual implementation will extend `framework-expansion`;
  this change records the target requirement so the design is traceable. No code
  or behavior changes here.
- **Behavior:** none (design only).
- **Risk:** none in this change; the point is to de-risk implementation by
  settling the model and its decisions first.

## Non-goals (of this design change)

- Implementing any of the model — the deliverable is the design + decisions.
- Changing the current builtin-only behavior.

```

## openspec/changes/framework-ecosystem-model/design.md

- Source: openspec/changes/framework-ecosystem-model/design.md
- Lines: 1-136
- SHA256: 03ddaefec4156c4a79f90e5e0018adc2656dd1a11aca9d92a8fea8b92097b22d

[TRUNCATED]

```md
# Design — framework ecosystem model (E1)

Design-only. Deliverable: the target architecture + the maintainer decisions.

## 1. Current model (baseline, verified in code)

- Manifest `frameworks/<name>/framework.toml`: `name`, `version` (content),
  `description`, `[dependencies].frameworks = [names]`, and `[skills]`/
  `[commands]`/`[subagents]` name→catalog-path maps (`internal/catalog/catalog.go`).
- `catalog.Load`: name==dir, every resource path exists, a resource name mapped
  to two *different* paths is a hard error; loose (frameworkless) resources are
  indexed by base name. `version.txt` is the catalog version.
- `expandResources` (`expand.go`): transitive dependency walk with white/grey/
  black **cycle detection** and an "unknown framework" error for a missing dep;
  first-wins collapse when a resource is reachable via two frameworks.
- Only embedded/builtin frameworks; `config.Load` rejects a non-builtin
  `[frameworks.X]` source (F35).

So **dependencies + cycles + duplicate-path conflict + builtin validation
already exist.** The gaps are schema-versioning, capabilities, compatibility,
local resolution, explicit conflict policy, and migrations.

## 2. Target manifest (proposed additive schema, v2)

```toml
manifest_schema = 2                 # NEW: manifest format version (fail-closed if newer)
name = "comet"
version = "0.2.0"                   # framework CONTENT version (semver)
description = "..."

[compat]                            # NEW
homonto = ">=0.1.0 <0.2.0"          # homonto version range this framework supports

[provides]                          # NEW: capabilities this framework offers
capabilities = ["spec-workflow@1"]  # name@major

[dependencies]
frameworks = ["superpowers@>=0.1.0"] # dep may pin a version RANGE (back-compat: bare name = any)
capabilities = ["planning@1"]        # NEW: depend on a capability, resolved to a provider

[skills]   # unchanged
[commands] # unchanged
[subagents]# unchanged
```

Every NEW field is optional; a v1 manifest (no `manifest_schema`, bare dep
names, no compat/provides) loads exactly as today.

## 3. Resolution / validation pipeline (target)

Extend `catalog.Load` (now phased like `config.Load`: decode → validate →
index) and `expandResources`:

1. **decode + manifest-version guard** — reject `manifest_schema` > supported,
   fail-closed ("upgrade homonto"), mirroring config/state schema versions.
2. **compat check** — reject a framework whose `[compat].homonto` excludes the
   running homonto version; reject a dependency whose resolved version is outside
   the declared range.
3. **capability resolution** — build a provider index (capability → framework);
   a `dependencies.capabilities` entry resolves to a provider (error if none, or
   ambiguous under the conflict policy).
4. **conflict policy** — make the current implicit policy explicit and named
   (see §4); apply to duplicate resources AND duplicate capability providers.
5. **existing checks** — path existence, name==dir, cycle detection (kept).

## 4. Conflict policy (decision)

Today: same-name→different-path = error; same path = first-wins. Options for the
explicit policy the exit gate wants:
- **(a) strict** (default): a resource/capability provided by two frameworks in
  the resolved set is an error unless identical — safest, matches today's
  path-conflict behavior.
- **(b) priority**: an explicit framework order (config-declared) breaks ties.
- **(c) override**: a config `[frameworks.X].overrides = [...]` allows an
  intentional shadow.
Recommendation: ship (a) strict as the model's default; add (b)/(c) only when a
real multi-framework conflict demands it.

## 5. Local/custom framework resolution (decision — the crux)


```

Full source: openspec/changes/framework-ecosystem-model/design.md

## openspec/changes/framework-ecosystem-model/tasks.md

- Source: openspec/changes/framework-ecosystem-model/tasks.md
- Lines: 1-21
- SHA256: 79500c2916f062885ac4d876fcb52efd0e2dd00e8ea89194d20b83fd366c687d

```md
# Tasks — framework-ecosystem-model (design + MVP)

## 1. Architecture design
- [x] Produce the target manifest schema (additive v2), the resolution/validation
      pipeline, capability + compatibility model, local-source trust reuse,
      explicit conflict policy, and a phased MVP→full delivery plan.

## 2. Surface decisions
- [x] Enumerate the blocking maintainer decisions (D1 local frameworks, D2
      capabilities, D3 conflict policy, D4 F38, D5 first-impl scope) with a
      recommendation for each (in design.md / the Design Doc).

## 3. MVP implementation (D-independent, phase 1)
- [ ] Add `manifest_schema` to the framework manifest + a fail-closed guard in
      catalog.Load (reject a manifest whose schema exceeds the supported version,
      "upgrade homonto"), mirroring the config/state schema-version pattern.
      Pure additive; every builtin manifest (no field / schema 1) loads
      unchanged. TDD: a future manifest_schema is rejected; absent/current load.

## 4. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/framework-ecosystem-model/specs/framework-expansion/spec.md

- Source: openspec/changes/framework-ecosystem-model/specs/framework-expansion/spec.md
- Lines: 1-32
- SHA256: 69a7f9d082e7cc43ed992e4f94c84812cf66919f89aeef9f7e299c082d4c595b

```md
# framework-expansion

## ADDED Requirements

### Requirement: The framework model supports versioned manifests and validated custom-source resolution

The framework ecosystem SHALL support versioned framework manifests and a single
validated resolution path that a builtin, a fourth builtin, or a trusted custom
framework all pass through. A framework manifest MAY declare a manifest schema
version, provided/required capabilities, and compatibility ranges; loading MUST
reject a manifest whose schema version exceeds what the binary supports (fail
closed), and MUST reject an incompatible framework or an unresolved required
capability with a clear error rather than silently installing nothing. The
existing guarantees — transitive dependency resolution, cycle detection, and
duplicate-resource rejection — MUST be preserved.

This requirement is recorded as the design target for roadmap E1; the design is
delivered and reviewed before implementation, which lands in later phased changes.

#### Scenario: A manifest from a newer schema is rejected

- **WHEN** a framework manifest declares a manifest schema version greater than
  the binary supports
- **THEN** loading fails closed with an "upgrade homonto" error and installs
  nothing

#### Scenario: A custom framework resolves through the same validated path

- **WHEN** a trusted custom framework is resolved
- **THEN** it is loaded and validated through the same manifest/dependency/
  path checks as a builtin framework, and an unsupported source or an
  incompatible version fails loudly

```
