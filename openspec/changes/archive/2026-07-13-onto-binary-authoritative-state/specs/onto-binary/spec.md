# onto-binary (delta)

## MODIFIED Requirements

### Requirement: onto-state.yaml change-state model

The `onto` binary SHALL read, validate, and write a per-change state file named
`onto-state.yaml` (at `docs/changes/<name>/onto-state.yaml`) through a dedicated
state package, as the single authority for onto workflow state. The model SHALL
parse the file into a typed structure carrying an explicit `schema_version`, a
typed **core** of gated fields, and a carried **observational** group that is
never gated. It SHALL validate the presence and shape of gated fields only
(enum/format), never their substantive quality (B1: the binary rejects a
malformed value, not an unconvincing one). It SHALL derive the current workflow
phase from the core.

The gated core SHALL include at least: change, workflow (`full|fix|tweak`), phase
(`open|design|build|verify|close`), created, base_ref, deps, isolation
(`branch|worktree|""`), build_mode (`direct|subagent|""`), tdd_mode
(`tdd|direct|""`), verify scale (`light|full|""`), verify result
(`pending|pass|fail`), close merged (bool), archived (bool), and the directive
string. Observational fields (metrics, task counts, verify rounds, escalation
flag) SHALL be carried through reads and writes but SHALL never gate a
transition. Writes SHALL always emit the current `schema_version`.

The binary SHALL migrate legacy state on read: a legacy binary `onto-state.yaml`
(no `schema_version`) and a legacy skill `state.yaml` (no `schema_version`) SHALL
each up-migrate to the current schema. Migration SHALL be ordered and idempotent
(loading a current-version file is a no-op). If a change directory holds BOTH a
legacy `onto-state.yaml` and a legacy `state.yaml` whose gated core fields
disagree (phase, workflow, or archived), the state SHALL be reported as malformed
rather than silently resolved. Parsing an invalid or malformed state SHALL return
a clear error identifying the file, not a panic.

The recognized workflow phases are `open`, `design`, `build`, `verify`, `close`,
with `close` as the terminal phase and `archived` as a terminal boolean.

#### Scenario: parse and derive phase from a valid versioned onto-state.yaml

- **GIVEN** a valid `onto-state.yaml` carrying `schema_version`, a gated core, and observational fields
- **WHEN** the state model loads it
- **THEN** it returns the typed state and the derived phase without error, preserving observational fields

#### Scenario: legacy state migrates on read

- **GIVEN** a legacy `onto-state.yaml` (no `schema_version`) or a legacy `state.yaml` (no `schema_version`)
- **WHEN** the state model loads it
- **THEN** it up-migrates to the current schema without dropping any gated field, and a subsequent write emits the current `schema_version`

#### Scenario: disagreeing dual legacy files are malformed

- **GIVEN** a change directory holding both a legacy `onto-state.yaml` and a legacy `state.yaml` whose phase, workflow, or archived disagree
- **WHEN** the state model loads the change
- **THEN** it reports the state as malformed and names the conflict, and does not silently pick a winner

#### Scenario: malformed state reports a clear error

- **GIVEN** a state file that is not valid YAML or fails presence/shape validation
- **WHEN** the state model loads it
- **THEN** it returns an error naming the file and the problem, and does not panic

### Requirement: onto status is read-only and config-independent

`onto status` SHALL be a read-only diagnostic command that inspects an existing
`docs/` workspace WITHOUT requiring a `homonto.toml` file or a declared
`[frameworks.onto]` entry. It SHALL enumerate change **directories** under
`docs/changes/` (excluding `archive/`) FIRST, then classify each as `valid`
(state present, parses, validates — report its derived phase), `malformed` (state
present but unparseable/invalid), or `missing-state` (a change directory with no
state file). A change directory whose state file was deleted SHALL therefore
appear as a `missing-state` row and SHALL NOT silently disappear. `onto status`
SHALL NOT create, modify, or delete any file.

#### Scenario: status classifies each change directory

- **GIVEN** `docs/changes/` with one valid change, one whose `onto-state.yaml` is malformed, and one directory with no state file
- **WHEN** `onto status` runs
- **THEN** it reports the first as `valid` with its phase, the second as `malformed`, and the third as `missing-state`, and exits without writing any file

#### Scenario: a deleted state file is not silently dropped

- **GIVEN** a change directory that once had `onto-state.yaml` but the file was deleted
- **WHEN** `onto status` runs
- **THEN** the directory is reported as `missing-state`, not omitted

#### Scenario: status leaves the worktree untouched

- **WHEN** `onto status` runs against any workspace
- **THEN** no file under `docs/` or elsewhere is created, modified, or removed

### Requirement: onto doctor reports workflow and project health

`onto doctor [--dir <root>]` SHALL be a strictly read-only, config-independent
diagnostic that reports the health of an onto workspace. It SHALL perform zero
writes, never construct a homonto config/engine, and never read `homonto.toml`.
It SHALL run regardless of whether the onto framework is installed. `--dir` SHALL
default to `.`.

`onto doctor` SHALL check, and surface each problem it finds as an individual
finding line:

- **docs layout**: `docs/changes`, `docs/specs`, `docs/adr`, and `docs/guides`
  each exist as directories under the root; a missing one is a finding.
- **active change classification**: it SHALL enumerate change **directories**
  under `docs/changes/` (the single `*` excludes archived changes) FIRST, then
  classify each as `valid` (state loads, validates, derives a phase), `malformed`
  (state present but unparseable/invalid), or `missing-state` (a change directory
  with no state file). A `malformed` or `missing-state` directory is a finding —
  a change whose state file was deleted SHALL be reported, not silently skipped.
- **phase matches artifacts**: for each valid active change, every artifact
  required for its derived phase is present; a missing required artifact is a
  finding.
- **dependency and gate consistency**: for each valid active change, every
  dependency it lists is resolved (an archived `docs/changes/archive/*-<dep>`
  exists); an unresolved dependency is a finding. An active change whose state
  records `archived: true` is a finding.
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

#### Scenario: a missing-state change directory is a finding

- **GIVEN** a change directory under `docs/changes/` that has no state file (e.g. `onto-state.yaml` was deleted)
- **WHEN** `onto doctor` runs
- **THEN** it classifies the directory as `missing-state`, reports it as a finding, and exits non-zero — the directory is not silently skipped

#### Scenario: invalid state is a finding

- **GIVEN** an active change whose state is malformed or fails validation
- **WHEN** `onto doctor` runs
- **THEN** it reports the change as invalid, naming the problem, and exits non-zero

#### Scenario: onto doctor is read-only and needs no framework install

- **GIVEN** a root with no `homonto.toml` and no installed onto framework
- **WHEN** `onto doctor` runs
- **THEN** it still runs (reporting docs-layout findings) and writes nothing

## ADDED Requirements

### Requirement: onto exposes state transitions and a structured read

The `onto` binary SHALL expose, through its CLI, a command for every gated state
mutation of an active change and a structured read of a change's full state, so a
caller can drive the entire workflow lifecycle without editing a state file by
hand. Each transition command SHALL validate the presence and shape of the field
it sets (rejecting a malformed value with a clear error) and SHALL write through
the versioned state model. The structured read SHALL emit the full validated
state (including derived phase) as JSON.

#### Scenario: a transition command sets a gated field with validation

- **GIVEN** an active change at a phase where a gated field (e.g. isolation) may be set
- **WHEN** the corresponding `onto` transition command runs with a valid value
- **THEN** the field is written through the state model and a subsequent read reflects it
- **AND** running it with a value outside the field's allowed shape is rejected with a clear error and no write

#### Scenario: structured read emits the full state as JSON

- **GIVEN** a change with a valid `onto-state.yaml`
- **WHEN** the `onto` structured read command runs for that change
- **THEN** it emits the full validated state and derived phase as JSON, writing no file
