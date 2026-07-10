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
