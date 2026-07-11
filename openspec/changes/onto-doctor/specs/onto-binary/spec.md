## ADDED Requirements

### Requirement: onto doctor reports workflow and project health

`onto doctor [--dir <root>]` SHALL be a strictly read-only, config-independent
diagnostic that reports the health of an onto workspace. It SHALL perform zero
writes, never construct a homonto config/engine, and never read `homonto.toml`.
It SHALL run regardless of whether the onto framework is installed (it is a
diagnostic, not a mutation, and is therefore NOT behind the framework-install
gate). `--dir` SHALL default to `.`.

`onto doctor` SHALL check, and surface each problem it finds as an individual
finding line:

- **docs layout**: `docs/changes`, `docs/specs`, `docs/adr`, and `docs/guides`
  each exist as directories under the root; a missing one is a finding.
- **active change state validity**: for each `docs/changes/*/onto-state.yaml`
  (the single `*` excludes archived changes, which live one level deeper), the
  state loads, validates, and derives a phase; a malformed or invalid file is a
  finding.
- **phase matches artifacts**: for each valid active change, every artifact
  required for its derived phase is present; a missing required artifact is a
  finding.
- **dependency and gate consistency**: for each valid active change, every
  dependency it lists is resolved (an archived `docs/changes/archive/*-<dep>`
  exists); an unresolved dependency is a finding. An active change whose state
  already records `archived: true` is a finding (an archived change belongs
  under `docs/changes/archive/`).
- **archive layout**: each `docs/changes/archive/*` entry is a directory holding
  a valid `onto-state.yaml` marked `archived: true`; a missing or invalid state
  file, or one not marked archived, is a finding.

On a healthy workspace `onto doctor` SHALL print a single `healthy` line and
exit 0. When one or more findings exist it SHALL print each finding and a count
summary and exit non-zero.

#### Scenario: healthy workspace reports healthy and exits 0

- **GIVEN** a root with the full `docs/{changes,specs,adr,guides}` layout, a valid active change whose artifacts match its phase and whose deps are resolved, and a well-formed archive entry
- **WHEN** `onto doctor` runs
- **THEN** it prints `healthy` and exits 0

#### Scenario: missing docs layout directory is a finding

- **GIVEN** a root missing `docs/adr`
- **WHEN** `onto doctor` runs
- **THEN** it reports the missing `docs/adr` directory and exits non-zero

#### Scenario: invalid onto-state.yaml is a finding

- **GIVEN** an active change whose `onto-state.yaml` is malformed or fails validation
- **WHEN** `onto doctor` runs
- **THEN** it reports the change as invalid, naming the problem, and exits non-zero

#### Scenario: phase not matching artifacts is a finding

- **GIVEN** an active change at a phase whose required artifacts are not all present (e.g. phase `build` without `plan.md`)
- **WHEN** `onto doctor` runs
- **THEN** it reports the missing required artifact and exits non-zero

#### Scenario: unresolved dependency is a finding

- **GIVEN** an active change whose `onto-state.yaml` lists a dependency that is not archived
- **WHEN** `onto doctor` runs
- **THEN** it reports the unresolved dependency and exits non-zero

#### Scenario: active change marked archived is a finding

- **GIVEN** an active change (under `docs/changes/<name>/`, not the archive) whose state records `archived: true`
- **WHEN** `onto doctor` runs
- **THEN** it reports the inconsistency and exits non-zero

#### Scenario: malformed archive entry is a finding

- **GIVEN** a `docs/changes/archive/<entry>` whose `onto-state.yaml` is missing, invalid, or not marked `archived: true`
- **WHEN** `onto doctor` runs
- **THEN** it reports the malformed archive entry and exits non-zero

#### Scenario: onto doctor is read-only and needs no framework install

- **GIVEN** a root with no `homonto.toml` and no installed onto framework
- **WHEN** `onto doctor` runs
- **THEN** it still runs (reporting docs-layout findings) and writes nothing — no file is created or modified
