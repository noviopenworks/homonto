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
