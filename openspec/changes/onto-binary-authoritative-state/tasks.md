# Tasks — onto-binary-authoritative-state

Open-phase task outline. The design phase resolves the deferred decisions; the
build phase turns these into a detailed plan.

## 1. Versioned state schema
- [ ] Define the versioned schema (superset of gated control fields + carried
      observational fields) with an explicit `schema_version`.
- [ ] Split gated vs observational fields per the B1 line; validate presence/shape
      of gated fields only.
- [ ] Round-trip (marshal→parse) tests including every gated field.

## 2. Migration from both legacy shapes
- [ ] Loader recognizes legacy `onto-state.yaml` (7-field), legacy
      `docs/changes/<name>/state.yaml` (rich), and the new versioned schema.
- [ ] Ordered, idempotent up-migration to the current version on read; writes
      always emit current version.
- [ ] Conflict policy for a dir carrying both legacy files.
- [ ] Migration tests over real `state.yaml` fixtures — assert no gated field is
      dropped.

## 3. CLI transition + read surface
- [ ] Every state mutation the skills do by hand has a binary command (extend
      `init/new/advance/close`; add the missing gated-field transitions).
- [ ] Structured (JSON) read command so callers query state without parsing files.
- [ ] Tests per command (happy path + validation rejection).

## 4. status/doctor enumerate + classify
- [ ] Enumerate change directories first, then classify valid / malformed /
      missing-state.
- [ ] A deleted state file appears as a `missing-state` row (F14 regression test).

## 5. Spec + verification
- [ ] Update `openspec/specs/onto-binary/spec.md` (delta) for the versioned
      schema, command surface, and classify behavior.
- [ ] `go test ./internal/ontostate/... ./internal/ontocli/...` green under -race.
- [ ] `go build ./...`, `go vet`, `openspec validate --all` green.

## 6. Confirm change B is ready to author
- [ ] Record the final schema + CLI surface so `onto-skills-shell-out` (change B)
      can be authored against concrete commands.
