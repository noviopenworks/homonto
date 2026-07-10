# onto-binary Specification

## Purpose
TBD - created by archiving change onto-binary-foundation. Update Purpose after archive.
## Requirements
### Requirement: Onto binary builds independently of homonto

The repository SHALL build a second binary `onto` from a dedicated
`package main` at `cmd/onto/`, via `go build ./cmd/onto` and installable with
`go install github.com/noviopenworks/homonto/cmd/onto`. The existing root
`homonto` binary (built from `main.go`) SHALL be unchanged, and `go build ./...`
SHALL build both.

#### Scenario: onto binary compiles from its own package main

- **GIVEN** the repository at a clean checkout
- **WHEN** `go build ./cmd/onto` runs
- **THEN** it produces an `onto` executable, and `go build ./...` still builds the `homonto` binary unchanged

### Requirement: Onto CLI root and version

The `onto` binary SHALL expose a Cobra root command `onto` constructed in the
same style as `homonto`'s `internal/cli.NewRootCmd`, with a `version` subcommand
that prints the build version. The version SHALL be a package-level variable
stampable at release time via `-ldflags "-X …Version=<tag>"`, mirroring how
`homonto`'s version is stamped.

#### Scenario: onto version prints the stamped version

- **WHEN** `onto version` runs
- **THEN** it prints `onto <version>` and exits 0
- **AND** a release build with `-ldflags "-X …Version=v0.1.0-rc.1"` prints that tag

### Requirement: onto-state.yaml change-state model

The `onto` binary SHALL read and validate a per-change state file named
`onto-state.yaml` through a dedicated state package. The model SHALL parse the
file into a typed structure, validate its structural fields, and derive the
current workflow phase from those fields. The file name is exactly
`onto-state.yaml`; there SHALL be no migration or back-compatibility layer for
the legacy `state.yaml` name (pre-release). Parsing an invalid or malformed
`onto-state.yaml` SHALL return a clear error identifying the file, not a panic.

The recognized workflow phases are `open`, `design`, `build`, `verify`, `close`
(the onto workflow phase set, matching the `onto-*` skills and the legacy
`state.yaml`), with `close` as the terminal phase.

#### Scenario: parse and derive phase from a valid onto-state.yaml

- **GIVEN** a valid `onto-state.yaml` recording a change's phase (one of open|design|build|verify|close) and gate fields
- **WHEN** the state model loads it
- **THEN** it returns the typed state and the derived phase without error

#### Scenario: malformed onto-state.yaml reports a clear error

- **GIVEN** an `onto-state.yaml` that is not valid YAML or is missing required fields
- **WHEN** the state model loads it
- **THEN** it returns an error naming the file and the problem, and does not panic

### Requirement: onto status is read-only and config-independent

`onto status` SHALL be a read-only diagnostic command that inspects an existing
`docs/` workspace and its `onto-state.yaml` files WITHOUT requiring a
`homonto.toml` file or a declared `[frameworks.onto]` entry (the read-only
degraded exception). It SHALL report each discovered change's derived phase and
flag any change whose state file is missing or invalid. `onto status` SHALL NOT
create, modify, or delete any file.

#### Scenario: status inspects a workspace without config

- **GIVEN** a project with `docs/changes/<name>/onto-state.yaml` but no `homonto.toml` and no `[frameworks.onto]`
- **WHEN** `onto status` runs
- **THEN** it reports each change's derived phase and exits 0 without writing any file

#### Scenario: status flags an invalid state file

- **GIVEN** a change whose `onto-state.yaml` is missing or malformed
- **WHEN** `onto status` runs
- **THEN** it reports that change as invalid/unreadable and still does not write any file

#### Scenario: status leaves the worktree untouched

- **WHEN** `onto status` runs against any workspace
- **THEN** no file under `docs/` or elsewhere is created, modified, or removed (read-only)

### Requirement: onto init scaffolds the workflow layout

`onto init` SHALL scaffold the onto workflow directory layout under the
workspace root: `docs/changes/`, `docs/specs/`, `docs/adr/`, and `docs/guides/`.
It SHALL be idempotent — an existing directory or file is preserved and never
overwritten — and it SHALL report which paths it created versus skipped. `onto
init` SHALL NOT create `homonto.toml` (that is `homonto init`'s job) and SHALL
NOT run the Homonto projection engine.

#### Scenario: init creates the docs layout in a prepared workspace

- **GIVEN** a workspace whose `homonto.toml` declares `[frameworks.onto]` and whose onto framework has been applied by Homonto
- **WHEN** `onto init` runs
- **THEN** `docs/changes/`, `docs/specs/`, `docs/adr/`, and `docs/guides/` exist and the command reports the created paths, exiting 0

#### Scenario: init is idempotent

- **GIVEN** a workspace where `onto init` already created the layout (and a user has added content under `docs/`)
- **WHEN** `onto init` runs again
- **THEN** existing directories and files are left untouched, newly-created paths (if any) are reported as created and pre-existing ones as skipped, and the command exits 0

### Requirement: onto init requires the Homonto-managed framework install

`onto init` is a mutating command and SHALL require that the project has declared
and applied `onto` through Homonto before it creates any `docs/` files:

- If `homonto.toml` is absent at the workspace root, `onto init` SHALL print a
  message directing the user to run `homonto init`, and exit non-zero.
- If `homonto.toml` exists but does not declare `[frameworks.onto]`, `onto init`
  SHALL print a message directing the user to declare `[frameworks.onto]` and run
  `homonto apply`, and exit non-zero.
- If `[frameworks.onto]` is declared but the onto framework has not been applied
  (no materialized evidence such as `.homonto/catalog/skills/onto/`), `onto init`
  SHALL print a message directing the user to run `homonto apply`, and exit
  non-zero.

In every failing case `onto init` SHALL NOT create, modify, or delete any file
under `docs/`.

#### Scenario: init refuses without homonto.toml

- **GIVEN** a workspace with no `homonto.toml`
- **WHEN** `onto init` runs
- **THEN** it prints guidance to run `homonto init`, creates no `docs/` files, and exits non-zero

#### Scenario: init refuses when frameworks.onto is not declared

- **GIVEN** a `homonto.toml` that does not declare `[frameworks.onto]`
- **WHEN** `onto init` runs
- **THEN** it prints guidance to declare `[frameworks.onto]` and run `homonto apply`, creates no `docs/` files, and exits non-zero

#### Scenario: init refuses when the framework is declared but not applied

- **GIVEN** a `homonto.toml` declaring `[frameworks.onto]` but no applied evidence (no `.homonto/catalog/skills/onto/`)
- **WHEN** `onto init` runs
- **THEN** it prints guidance to run `homonto apply`, creates no `docs/` files, and exits non-zero

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

### Requirement: Per-phase required artifacts

`internal/ontostate.RequiredArtifacts(phase)` SHALL return the cumulative set of
files that must exist at each workflow phase: `open` → `onto-state.yaml`,
`proposal.md`, `tasks.md`; `design` → those plus `design.md`; `build` → those
plus `plan.md`; `verify` and `close` → those plus `verification.md`. An unknown
phase SHALL return at least the `open` base set. `ValidateSkeleton` SHALL use this
per-phase set so a change's skeleton requirements tighten as it advances.

#### Scenario: build phase requires plan.md

- **GIVEN** a change at phase `build` with `onto-state.yaml`, `proposal.md`, `tasks.md`, `design.md` but no `plan.md`
- **WHEN** `ValidateSkeleton` runs
- **THEN** it returns an error naming `plan.md` as missing

### Requirement: onto advance gates phase transitions

`onto advance <change>` SHALL move a change to the next phase in the fixed order
`open → design → build → verify → close`, and ONLY through that order (no skips,
no reversals). It SHALL run the framework-install gate first. Before advancing it
SHALL verify the transition's precondition, which is that the CURRENT phase's
deliverables are complete (a phase's artifacts are produced while a change is in
that phase, so they gate leaving it, not entering it):

- every artifact in `RequiredArtifacts(currentPhase)` exists (e.g. leaving
  `design` requires `design.md`; leaving `build` requires `plan.md`; leaving
  `verify` requires `verification.md`; leaving `open` requires only the open
  artifacts proposal.md + tasks.md), AND
- when leaving `build`, every `tasks.md` checkbox is checked (at least one
  checkbox present, no unchecked `- [ ]`).

On success it SHALL write the new phase to `onto-state.yaml` and report the
transition. On a failed precondition it SHALL exit non-zero, name what is
missing, and leave the recorded phase unchanged. Advancing a change already at
`close` (or with an unknown phase) SHALL be an error with no write.

#### Scenario: advance open to design needs only the open artifacts

- **GIVEN** a change at phase `open` with `proposal.md` and `tasks.md` (as `onto new` creates), and no `design.md` yet
- **WHEN** `onto advance <change>` runs
- **THEN** the recorded phase becomes `design` and the command reports `open → design`, exiting 0

#### Scenario: advance refuses when the current phase's deliverable is missing

- **GIVEN** a change at phase `design` that has not yet produced `design.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero naming `design.md` as missing and the recorded phase stays `design`

#### Scenario: advance out of build requires all tasks checked

- **GIVEN** a change at phase `build` with `plan.md` present but an unchecked `- [ ]` item in `tasks.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating tasks are incomplete and the recorded phase stays `build`

#### Scenario: advance past close is an error

- **GIVEN** a change at phase `close`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating the change is already at the terminal phase and writes nothing

### Requirement: dirty worktree blocks the close transition

`onto advance` SHALL check the workspace's git worktree cleanliness (via `git
status --porcelain`). A dirty worktree SHALL produce a WARNING for a normal
transition (open→design, design→build, build→verify) but SHALL still allow it.
For the release-critical `verify → close` transition a dirty worktree SHALL BLOCK
the advance: the command exits non-zero, reports the dirty worktree, and does not
change the phase.

#### Scenario: dirty worktree warns but allows a normal advance

- **GIVEN** a change at phase `open` (with the open artifacts proposal.md + tasks.md) in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it advances to `design` (exit 0) after printing a dirty-worktree warning

#### Scenario: dirty worktree blocks verify to close

- **GIVEN** a change at phase `verify` whose `verification.md` (and earlier deliverables) exist, in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero reporting the dirty worktree and the recorded phase stays `verify`
