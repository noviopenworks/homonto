## ADDED Requirements

### Requirement: homonto agents doctor reports agent health

`homonto agents doctor` SHALL be a read-only command that loads the config
(declared agents) and `.homonto/agents-lock.json` (installed agents) and reports
each drift as a finding. It SHALL write nothing. It SHALL check:

- a declared agent absent from the lockfile is **not installed**;
- a lockfile-recorded agent absent from the config is **orphaned**;
- a `local:` agent whose `homonto/agents/<source>.md` content hash differs from
  the recorded install hash (or whose source file is missing) has a **source
  drift**;
- a target the agent declares but has no lockfile install entry is a **target not
  installed**;
- a lockfile install entry for a target the agent no longer declares is a
  **target no longer declared**;
- a recorded install path that no longer exists on disk is **missing on disk**;
- a `copy`-mode install whose on-disk content hash differs from the recorded hash
  is **modified on disk**.

On a healthy workspace it SHALL print `healthy` and exit 0. When one or more
findings exist it SHALL print each finding and exit non-zero.

#### Scenario: healthy workspace

- **GIVEN** a config whose declared agents are all installed, undrifted, and unmodified per the lockfile and disk
- **WHEN** `homonto agents doctor` runs
- **THEN** it prints `healthy` and exits 0

#### Scenario: declared but not installed

- **GIVEN** a `[agents.<name>]` with no lockfile record
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the agent as not installed and exits non-zero

#### Scenario: orphaned install

- **GIVEN** a lockfile agent that is no longer declared in the config
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the agent as orphaned and exits non-zero

#### Scenario: source drift

- **GIVEN** an installed `local:` agent whose source file content changed since install
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the source drift and exits non-zero

#### Scenario: modified on disk

- **GIVEN** a copy-mode installed agent whose on-disk file content was edited
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the file as modified on disk and exits non-zero

#### Scenario: missing on disk

- **GIVEN** a recorded install whose file was deleted
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the file as missing on disk and exits non-zero

#### Scenario: read-only

- **WHEN** `homonto agents doctor` runs
- **THEN** it writes no files and mutates nothing
