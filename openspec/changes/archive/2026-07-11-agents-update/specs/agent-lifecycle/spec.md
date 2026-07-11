## ADDED Requirements

### Requirement: homonto agents update re-materializes an installed agent

`homonto agents update <name>` SHALL re-install an already-installed declared
agent from its current source and refresh `.homonto/agents-lock.json`. The agent
MUST be declared and recorded in the lockfile; an undeclared or not-yet-installed
agent SHALL be an error (the latter directing the user to `agents add`). This
increment supports `local:<x>` sources only; `builtin:`/remote sources SHALL
return a clear "not yet supported" error.

For each of the agent's declared targets the command SHALL re-materialize per the
agent's mode: `copy` writes the current `homonto/agents/<x>.md` content; `link`
ensures the symlink points at the source. It SHALL be:

- **backup-preserving**: before overwriting a `copy`-mode target whose on-disk
  content differs from the recorded hash (a local edit), the current file SHALL be
  copied to `<path>.bak`;
- **idempotent**: a target already matching the source SHALL be a no-op;
- **recorded**: the lockfile SHALL be refreshed with each target's new content
  hash.

#### Scenario: update re-materializes a changed source

- **GIVEN** an installed copy-mode `local:` agent whose source file content changed since install
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target file is rewritten to the new source content and the lockfile hash is refreshed

#### Scenario: update backs up a locally-modified install

- **GIVEN** an installed copy-mode agent whose on-disk file was edited (differs from the recorded hash)
- **WHEN** `homonto agents update <name>` runs
- **THEN** the current file is copied to `<path>.bak` before the source content overwrites it

#### Scenario: update is idempotent

- **GIVEN** an installed agent already matching its source
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target is a no-op and no `.bak` is created

#### Scenario: update requires a prior install

- **GIVEN** a declared agent with no lockfile record
- **WHEN** `homonto agents update <name>` runs
- **THEN** it errors that the agent is not installed and points to `agents add`

#### Scenario: builtin source is not yet supported

- **GIVEN** an installed-or-declared `builtin:` agent
- **WHEN** `homonto agents update <name>` runs
- **THEN** it returns a clear error that builtin sources are not yet supported
