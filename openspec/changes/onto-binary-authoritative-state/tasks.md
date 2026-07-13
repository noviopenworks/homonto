# Tasks — onto-binary-authoritative-state

Implemented via the 9-task plan `docs/superpowers/plans/2026-07-13-onto-binary-authoritative-state.md`
(TDD, executing-plans). All work landed on branch
`feature/20260713/onto-binary-authoritative-state`.

## 1. Versioned state schema
- [x] Define the versioned schema (gated core flat on `State` + carried nested
      `Observed`) with an explicit `schema_version` (=1). (plan Task 1)
- [x] Split gated vs observational fields per the B1 line; `Validate` checks
      presence/shape of gated fields only, never `Observed`. (plan Task 1)
- [x] Round-trip (marshal→parse) tests over a full rich fixture — every gated
      field preserved. (plan Task 1)

## 2. Migration from both legacy shapes
- [x] `parseAndMigrate` recognizes legacy `onto-state.yaml` (7-field), legacy
      `state.yaml` (rich), and the versioned schema. (plan Task 2)
- [x] Ordered, idempotent up-migration on read; writes always emit current
      version. (plan Task 2)
- [x] `LoadChange` conflict policy for a dir carrying both legacy files:
      disagreeing gated core (phase/workflow/archived) → malformed/fail-loud;
      else merge Observed. (plan Task 3)
- [x] Migration tests over rich-fixture — asserts every gated field maps, no
      drop. (plan Task 2/3)

## 3. CLI transition + read surface
- [x] `onto set {isolation,build-mode,tdd-mode,verify-scale,verify-result,
      close-merged,directive}` — a command per gated mutation. (plan Task 4/5)
- [x] `onto state <change> --json` structured read (writes nothing). (plan Task 6)
- [x] Tests per command (happy path + shape rejection). (plan Task 4/5/6)

## 4. status/doctor enumerate + classify
- [x] `onto status` and `onto doctor` enumerate change directories first, then
      classify valid / malformed / missing-state. (plan Task 7/8)
- [x] A deleted state file appears as a `missing-state` row/finding (F14
      regression tests in both). (plan Task 7/8)

## 5. Spec + verification
- [x] `openspec/specs/onto-binary/spec.md` delta authored (MODIFY state-model +
      status + doctor, ADD transitions+read) — synced at archive.
- [x] `go test ./internal/ontostate/... ./internal/ontocli/... -race` → 106 passed.
- [x] `go build ./...`, `go vet ./...`, `openspec validate --all` (16/16) green.

## 6. Confirm change B is ready to author
- [x] Concrete surface recorded for `onto-skills-shell-out` (change B):
      - State file: `docs/changes/<name>/onto-state.yaml`, `schema_version: 1`;
        gated core fields flat + nested `verify:`/`close:`/`observed:`.
      - Transitions: `onto set isolation|build-mode|tdd-mode|verify-scale|
        verify-result <change> <value>`, `onto set close-merged <change>`,
        `onto set directive <change> <text>`.
      - Read: `onto state <change> --json`.
      - Diagnostics: `onto status` / `onto doctor` classify
        `valid|malformed|missing-state`.
      Change B rewrites the `onto*` skills to invoke these instead of writing
      state by hand — NOT done here (NON-GOAL).
