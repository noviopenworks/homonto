# cli-commands (delta)

## ADDED Requirements

### Requirement: doctor reports a pending catalog upgrade

`homonto doctor` SHALL report a finding when the catalog version recorded in state
differs from the embedded catalog version, indicating a pending catalog upgrade
that `apply` would materialize, so a stale materialized catalog is visible rather
than silent.

#### Scenario: a catalog-version mismatch is reported

- **GIVEN** a recorded catalog version that differs from the embedded catalog version
- **WHEN** `homonto doctor` runs
- **THEN** it reports a finding naming the pending catalog upgrade and pointing at `apply`
