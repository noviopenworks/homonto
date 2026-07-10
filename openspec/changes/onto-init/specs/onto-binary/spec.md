## ADDED Requirements

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
