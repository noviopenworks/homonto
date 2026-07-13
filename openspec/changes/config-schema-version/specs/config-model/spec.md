# config-model

## ADDED Requirements

### Requirement: Config carries an explicit schema version and rejects newer ones

Config loading SHALL reject a configuration whose top-level integer
`schema_version` is greater than the version the running binary supports,
failing closed with an error that directs the user to upgrade homonto — before
any adapter, plan, or apply logic runs. An absent or zero `schema_version` MUST
be treated as a legacy configuration equal to the current version, so existing
configs load unchanged. This gives the config plane the same forward-safety as
the versioned state file: a newer config is never silently mis-applied by an
older binary.

#### Scenario: A config from a newer schema is rejected

- **WHEN** `homonto.toml` declares a `schema_version` greater than the binary's
  supported version
- **THEN** loading fails with an "upgrade homonto" error and no projection is
  computed

#### Scenario: A legacy or current config loads unchanged

- **WHEN** `homonto.toml` omits `schema_version` (or sets it to the current
  supported version)
- **THEN** it loads and projects exactly as before
