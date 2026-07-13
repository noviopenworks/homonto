# config-model (delta)

## ADDED Requirements

### Requirement: state.json carries a versioned schema

`homonto` state (`state.json`) SHALL carry an explicit `schemaVersion`. `homonto`
SHALL stamp the current schema version on every write, SHALL treat an absent or
zero `schemaVersion` as the current legacy version (backward compatible), and
SHALL reject at load a `schemaVersion` greater than the version it supports
(fail-closed on an unknown future format) with a clear error rather than
misinterpreting it.

#### Scenario: a future state schema version is rejected

- **GIVEN** a `state.json` whose `schemaVersion` exceeds the version this binary supports
- **WHEN** the state is loaded
- **THEN** it is rejected with a clear "unknown schema version" error, not read as the current format

#### Scenario: a legacy state without a version still loads

- **GIVEN** a `state.json` with no `schemaVersion` field
- **WHEN** the state is loaded
- **THEN** it loads as the current legacy version, and a subsequent write stamps the current `schemaVersion`
