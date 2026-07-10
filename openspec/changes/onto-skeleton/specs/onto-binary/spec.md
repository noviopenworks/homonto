## ADDED Requirements

### Requirement: onto-state.yaml writer

`internal/ontostate` SHALL provide a serializer that round-trips with its parser:
`Marshal(State) ([]byte, error)` producing YAML that `Parse` reads back to an
equal `State`, and `Save(path string, s State) error` writing that YAML
atomically (temp + rename). `Save` SHALL NOT clobber via a partial write on
error.

#### Scenario: state round-trips through Marshal and Parse

- **GIVEN** a valid `State` (change + phase build)
- **WHEN** it is `Marshal`ed and the bytes are `Parse`d back
- **THEN** the parsed `State` equals the original (change, phase, and any set fields)

### Requirement: onto new creates a change skeleton

`onto new <change-name>` SHALL create `docs/changes/<change-name>/` containing an
`onto-state.yaml` (`change` = the name, `workflow` defaulting to `full`, `phase`
= `open`, `created` = the current date) and empty-but-present `proposal.md` and
`tasks.md` skeleton files. It SHALL run the framework-install gate first (same as
`onto init`), SHALL validate `<change-name>` is kebab-case with no path traversal
(reject `..`, `/`, empty), and SHALL REFUSE with a non-zero exit and NO writes if
`docs/changes/<change-name>/` already exists (never clobber an existing change).

#### Scenario: new creates the open-phase skeleton

- **GIVEN** a prepared workspace (framework-install gate passes) with no `docs/changes/feature-x/`
- **WHEN** `onto new feature-x` runs
- **THEN** `docs/changes/feature-x/onto-state.yaml` (phase open), `proposal.md`, and `tasks.md` exist, and the command reports the created change, exiting 0

#### Scenario: new refuses to clobber an existing change

- **GIVEN** `docs/changes/feature-x/` already exists (with content)
- **WHEN** `onto new feature-x` runs
- **THEN** it exits non-zero, prints that the change already exists, and modifies no file under `docs/changes/feature-x/`

#### Scenario: new rejects an invalid change name

- **WHEN** `onto new "../evil"` (or a non-kebab-case / empty name) runs
- **THEN** it exits non-zero with a validation error and creates nothing

#### Scenario: new requires the framework install

- **GIVEN** a workspace without `homonto.toml` or `[frameworks.onto]` or the applied onto framework
- **WHEN** `onto new feature-x` runs
- **THEN** it prints the same framework-install guidance as `onto init`, creates nothing, and exits non-zero

### Requirement: phase-aware skeleton validation

`internal/ontostate` SHALL expose the artifacts required for each workflow phase
(`RequiredArtifacts(phase) []string`) and a `ValidateSkeleton(changeDir) error`
that confirms the files required for the change's recorded phase are present. For
the `open` phase the required artifacts SHALL be `onto-state.yaml`, `proposal.md`,
and `tasks.md`. `onto status` SHALL report each change's skeleton validity
(e.g. "skeleton ok" or "skeleton: missing <file>") without writing any file.

#### Scenario: status reports a complete open-phase skeleton as ok

- **GIVEN** a change at phase open with `onto-state.yaml`, `proposal.md`, `tasks.md`
- **WHEN** `onto status` runs
- **THEN** it reports the change's phase and that its skeleton is ok, writing nothing

#### Scenario: status reports a missing required artifact

- **GIVEN** a change at phase open missing `tasks.md`
- **WHEN** `onto status` runs
- **THEN** it reports the change's skeleton as missing `tasks.md`, still writing nothing and not aborting other changes
