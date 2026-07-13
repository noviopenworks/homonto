# onto-binary (delta)

## MODIFIED Requirements

### Requirement: onto new creates a change skeleton

`onto new <change-name> [--workflow full|fix|tweak]` SHALL create
`docs/changes/<change-name>/` containing an `onto-state.yaml` (`change` = the
name, `workflow` = the `--workflow` value defaulting to `full`, `phase` = `open`,
`created` = the current date) and empty-but-present `proposal.md` and `tasks.md`
skeleton files. `--workflow` SHALL accept only `full`, `fix`, or `tweak`; any
other value SHALL be rejected with a non-zero exit and no writes. It SHALL run the
framework-install gate first (same as `onto init`), SHALL validate `<change-name>`
is kebab-case with no path traversal (reject `..`, `/`, empty), and SHALL REFUSE
with a non-zero exit and NO writes if `docs/changes/<change-name>/` already exists.

#### Scenario: new creates the open-phase skeleton with the chosen workflow

- **GIVEN** a prepared workspace (framework-install gate passes) with no `docs/changes/feature-x/`
- **WHEN** `onto new feature-x --workflow fix` runs
- **THEN** `docs/changes/feature-x/onto-state.yaml` exists with `phase: open` and `workflow: fix`, alongside `proposal.md` and `tasks.md`, exiting 0

#### Scenario: new defaults workflow to full

- **WHEN** `onto new feature-y` runs with no `--workflow`
- **THEN** the created `onto-state.yaml` has `workflow: full`

#### Scenario: new rejects an invalid workflow

- **WHEN** `onto new feature-z --workflow epic` runs
- **THEN** it exits non-zero with a validation error and creates nothing

#### Scenario: new refuses to clobber an existing change

- **GIVEN** `docs/changes/feature-x/` already exists (with content)
- **WHEN** `onto new feature-x` runs
- **THEN** it exits non-zero, prints that the change already exists, and modifies no file under `docs/changes/feature-x/`

#### Scenario: new rejects an invalid change name

- **WHEN** `onto new "../evil"` (or a non-kebab-case / empty name) runs
- **THEN** it exits non-zero with a validation error and creates nothing

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
(`pending|pass|fail`), close merged (bool), guides
(`pending|updated|"waived: <reason>"|""`), archived (bool), and the directive
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

#### Scenario: guides accepts pending, updated, and waived forms

- **GIVEN** a change whose `guides` is being set
- **WHEN** it is set to `pending`, `updated`, or a value beginning `waived:`
- **THEN** the value is accepted; any other non-empty value is rejected with a clear error and no write

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

### Requirement: onto exposes state transitions and a structured read

The `onto` binary SHALL expose, through its CLI, a command for every gated state
mutation of an active change and a structured read of a change's full state, so a
caller can drive the entire workflow lifecycle without editing a state file by
hand. This SHALL include setters for isolation, build mode, tdd mode, verify scale,
verify result, close merged, directive, base ref, deps, and guides. Each
transition command SHALL validate the presence and shape of the field it sets
(rejecting a malformed value with a clear error) and SHALL write through the
versioned state model. The structured read SHALL emit the full validated state
(including derived phase) as JSON.

#### Scenario: a transition command sets a gated field with validation

- **GIVEN** an active change at a phase where a gated field (e.g. isolation) may be set
- **WHEN** the corresponding `onto` transition command runs with a valid value
- **THEN** the field is written through the state model and a subsequent read reflects it
- **AND** running it with a value outside the field's allowed shape is rejected with a clear error and no write

#### Scenario: base-ref and deps setters record creation fields

- **GIVEN** an active change
- **WHEN** `onto set base-ref <change> <ref>` and `onto set deps <change> --dep a --dep b` run
- **THEN** the state records the base ref and the dependency list, reflected in a subsequent read

#### Scenario: structured read emits the full state as JSON

- **GIVEN** a change with a valid `onto-state.yaml`
- **WHEN** the `onto` structured read command runs for that change
- **THEN** it emits the full validated state and derived phase as JSON, writing no file
