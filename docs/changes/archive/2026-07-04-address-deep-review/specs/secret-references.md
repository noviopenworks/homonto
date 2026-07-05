# Delta Spec: secret-references (address-deep-review)

## ADDED Requirements

### Requirement: Unknown-provenance redaction

Plan and log output SHALL treat as secret any managed key that exists on
disk but has no record in state: the old value SHALL be displayed only as
the redaction marker, never as the raw on-disk value — absence of
provenance is never grounds for printing a value in cleartext.

#### Scenario: Missing state does not leak

- **GIVEN** a secret-backed key already written to a tool file, and a
  missing (or key-less) `state.json`
- **WHEN** `homonto plan` shows an update for that key
- **THEN** the old value is rendered redacted (e.g. `«secret»`) and the
  on-disk secret never appears in the output

### Requirement: Secret-file modes

Files homonto writes SHALL preserve the existing file's permission mode
when the file already exists, default to `0600` for newly created files,
and be fsync'd before the atomic rename so a crash cannot leave an empty
or truncated secret-bearing file.

#### Scenario: New file created private

- **GIVEN** a tool file that does not yet exist and desired content
  containing a resolved secret
- **WHEN** apply writes it
- **THEN** the file is created with mode `0600`

#### Scenario: Existing mode preserved, never loosened

- **GIVEN** an existing tool file with mode `0600`
- **WHEN** apply rewrites it
- **THEN** the file still has mode `0600` after the write (the temp-file
  path does not reset it to a wider default)
