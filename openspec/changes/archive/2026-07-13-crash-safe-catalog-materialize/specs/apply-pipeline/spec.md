# apply-pipeline

## ADDED Requirements

### Requirement: Builtin-skill materialization is atomic per skill

Materializing a builtin skill's directory SHALL be atomic: the destination skill
directory MUST only ever contain a complete skill, never a partially-written
one. Implementations MUST write the skill's files to a staging location and swap
it into place only after all files are written, so that a read error, full disk,
or process crash during materialization leaves either the previous complete skill
directory or no directory at all — never a partial one that the re-materialize
gate would mistake for a complete skill.

#### Scenario: A failure mid-materialization does not corrupt the destination

- **WHEN** materializing a skill fails partway through writing its files
- **THEN** the destination skill directory is left in its prior complete state
  (or absent if it never existed), never partially written

#### Scenario: Successful materialization writes identical content

- **WHEN** a skill materializes successfully via stage-then-swap
- **THEN** the destination contains exactly the skill's files, byte-for-byte the
  same as a direct write
