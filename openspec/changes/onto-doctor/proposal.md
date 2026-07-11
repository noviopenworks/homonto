## Why

The onto workflow engine is complete: `onto new` creates, `onto advance`
transitions, and `onto close` archives a change. But nothing checks whether an
existing onto workspace is *healthy* — that its docs layout is intact, its
`onto-state.yaml` files are valid, each change's phase matches the artifacts it
should have, its dependencies are resolved, and its archive layout is
well-formed. The dual-binary design defines `onto doctor` as the peer to
`homonto doctor`: `homonto doctor` checks installation/projection health, while
`onto doctor` checks workflow and project health. This change (#4 of the onto
binary work) adds `onto doctor`.

## What Changes

- Add `onto doctor [--dir]`: a strictly read-only, config-independent diagnostic
  that reports the health of an onto workspace and exits non-zero if any problem
  is found (so CI and smoke tests can gate on it). It writes nothing, never
  constructs a homonto config/engine, and never reads `homonto.toml` — onto
  stays isolated from homonto's projection pipeline. It runs regardless of
  whether the onto framework is installed (it is a diagnostic, not a mutation,
  so it is **not** behind the framework-install gate that `init`/`new`/`close`
  use); a missing docs layout is reported as a finding, not a refusal.
- Checks performed (each surfaced as an individual finding line):
  - **docs layout**: `docs/changes`, `docs/specs`, `docs/adr`, `docs/guides`
    each exist as directories (reusing `onto init`'s `docsLayout`).
  - **active change state validity**: for each `docs/changes/*/onto-state.yaml`,
    it loads and validates the model and derives the phase; a malformed or
    invalid file is a finding.
  - **phase-derivation-matches-artifacts**: for each valid change,
    `ontostate.ValidateSkeleton` confirms every artifact
    `RequiredArtifacts(phase)` names is present; a missing artifact is a
    finding.
  - **gates and dependencies consistent**: for each valid change,
    `ontostate.DepsResolved` reports any unresolved dependency as a finding; an
    active (non-archived-directory) change whose state already records
    `archived: true` is a finding (an archived change belongs under
    `docs/changes/archive/`, not in the active set).
  - **archive layout valid**: each `docs/changes/archive/*` entry is a directory
    containing a valid `onto-state.yaml` marked `archived: true`; a missing or
    invalid state file, or one not marked archived, is a finding.
- On a healthy workspace it prints a single `healthy` summary and exits 0. When
  findings exist it prints each finding, a count summary, and exits non-zero.
- Register `doctorCmd()` on the onto root. `onto` binary commands become:
  **advance / close / doctor / init / new / status / version**.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains the `onto doctor` read-only workflow/project health
  command (docs-layout, change-state-validity, phase-vs-artifacts,
  dependency-consistency, and archive-layout checks; non-zero exit on findings).

## Impact

- `internal/ontocli`: new `doctor.go` (`doctorCmd()`/`runDoctor()`), registered
  on the root; reuses `docsLayout` and the existing `ontostate` API (`Load`,
  `Validate`, `DerivePhase`, `ValidateSkeleton`, `RequiredArtifacts`,
  `DepsResolved`). No new exported `ontostate` helper is required.
- No new dependency. `onto` stays isolated from homonto's projection pipeline
  (`internal/{cli,engine,config,adapter,catalog}` remain unimported by
  `internal/ontocli`).
- Read-only: `onto doctor` performs zero writes, mirroring `onto status`.
- Completes #4 of the onto binary work; only dual-binary release packaging (#5)
  remains.
