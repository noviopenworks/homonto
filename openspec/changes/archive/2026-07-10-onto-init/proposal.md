## Why

Change #1 (`onto-binary-foundation`, archived) shipped the `onto` binary with the
`onto-state.yaml` model and read-only `onto status`. This change (#2 of the
five-change `onto` decomposition) adds `onto init` — the first MUTATING command —
which scaffolds the `docs/` workflow layout. Per the dual-binary design, `onto`
is managed by Homonto: `onto init` may create the workspace only after the
project declares and applies `[frameworks.onto]` through Homonto; if that
framework install is missing, it directs the user to initialize/apply Homonto
first. This keeps `onto` a Homonto-managed operator, not an alternate installer.

## What Changes

- Add `onto init`: scaffolds the onto workflow layout
  `docs/{changes,specs,adr,guides}/` under the workspace root, idempotently
  (existing files/dirs are preserved, never clobbered), and reports what it
  created vs. skipped. It does NOT create `homonto.toml` (that is `homonto init`).
- **Framework-install gate** (mutating precondition): `onto init` requires (a) a
  `homonto.toml` at the workspace root declaring `[frameworks.onto]`, and (b) the
  onto framework materialized/installed by Homonto (evidence:
  `.homonto/catalog/skills/onto/` exists). If `homonto.toml` is absent or lacks
  `[frameworks.onto]`, or the framework is declared but not yet applied, `onto
  init` prints a clear message telling the user to run `homonto init` / declare
  `[frameworks.onto]` / run `homonto apply`, and exits non-zero WITHOUT creating
  any `docs/` files.
- `onto init` is additive to the `onto-binary` capability; `onto status`
  (read-only, config-independent) and the state model are unchanged.
- This change does NOT add phase-gate enforcement / skeleton create-validate
  (#3), `onto doctor` (#4), or release packaging (#5).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains the `onto init` command and the framework-install gate
  precondition for mutating operations. The foundation's read-only `onto status`
  and state model are unchanged.

## Impact

- New `internal/ontocli/init.go` (`initCmd()`), registered on the onto root.
- New gate helper — reads `homonto.toml` for `[frameworks.onto]` and checks the
  materialized onto framework. Prefer reusing `internal/config.Load` to parse the
  config and read `Config.Frameworks["onto"]`; check `.homonto/catalog/skills/onto`
  for the applied evidence. `onto init` must NOT run the projection engine.
- New `internal/ontocli/init_test.go`.
- No change to `homonto`, `internal/cli`, adapters, engine, config, catalog, or
  `internal/ontostate`.
- Advances the `onto` binary toward the dual-binary release gate (#2 of 5).
