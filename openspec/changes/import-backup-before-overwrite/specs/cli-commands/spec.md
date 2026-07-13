# cli-commands (delta)

## ADDED Requirements

### Requirement: import backs up an existing config before overwriting

`homonto import --force` SHALL, before overwriting an existing config file, copy
the existing file to `<config>.bak`, and SHALL write the new config atomically, so
a forced import over a valid config is recoverable and never leaves a partially
written file.

#### Scenario: forced import over an existing config preserves a backup

- **GIVEN** an existing `homonto.toml` with valid content
- **WHEN** the user runs `homonto import --force`
- **THEN** the previous content is preserved at `homonto.toml.bak` and the new config is written atomically
