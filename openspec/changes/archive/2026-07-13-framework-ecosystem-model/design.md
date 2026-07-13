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

F35 rejects every non-builtin `[frameworks.X]`. To lift that safely, reuse the
**already-shipped remote-source trust pipeline** (`internal/remote/`, digest-
pinned, verify-before-materialize, fail-closed — see the remote-source-trust
archive) that subagents already use. A local/custom framework would be:
- `source = "local:<path>"` — a directory validated through the SAME `catalog`
  loader (manifest-version, compat, capabilities, paths), OR
- `source = "remote:<url>"` + `digest` — fetched/verified/materialized through
  the remote pipeline, then loaded as a catalog overlay.
Open decisions: (i) local: trust — validate structurally but no digest (local
files are the user's own), vs require a digest for reproducibility; (ii) whether
custom frameworks merge into the builtin catalog namespace or stay isolated;
(iii) precedence when a custom framework redefines a builtin resource (→ §4).

## 6. Migrations (decision)

Manifest migrations matter only once external manifests exist at rest. For
builtins (shipped with the binary) there is never a version skew. Recommendation:
**defer migrations** until local/custom frameworks land; the manifest-version
*guard* (fail-closed) is the forward-safety that ships first.

## 7. F38 plugin lifecycle (decision)

`[plugins]` today is enable/disable projection (claude `enabledPlugins`,
opencode `plugin` array) — not install/update. Options: (a) build a real plugin
lifecycle (install/update/pin) — large; (b) **honest rename/doc** to
"plugin enablement" so the capability is not oversold. Recommendation: (b) now,
(a) only if a real install-lifecycle need appears.

## 8. Phased delivery (recommended MVP → full)

1. **MVP** (smallest valuable, verifiable): `manifest_schema` field +
   fail-closed guard in `catalog.Load` (forward-safety), and dependency *version
   ranges* validated against the resolved framework `version` (compat for deps).
   Pure additive; builtins unaffected.
2. `[compat].homonto` range check.
3. Capabilities (`provides`/`dependencies.capabilities`) + provider index +
   strict conflict policy made explicit.
4. Local/custom framework resolution via the remote-trust pipeline (the big one;
   needs the §5 decisions).
5. F38 honest rename; migrations if/when external manifests land.

## 9. Decisions the maintainer must make (blocking implementation)

- **D1 local frameworks**: allow `local:` (structural validation only) and/or
  `remote:`+digest (via the trust pipeline)? Trust model for each? (§5)
- **D2 capabilities**: adopt `name@major` capability strings, or defer
  capabilities entirely and keep name-based deps? (§2/§3)
- **D3 conflict policy**: strict-only (recommended), or also priority/override? (§4)
- **D4 F38**: real plugin lifecycle, or honest rename now? (§7)
- **D5 scope of first implementation**: just the MVP (§8.1), or through compat +
  capabilities (§8.1–3)?

## Alternatives considered

- A brand-new manifest format — rejected; the additive v2 keeps every existing
  builtin manifest valid and the change incremental.
