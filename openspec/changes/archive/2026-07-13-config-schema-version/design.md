# Design — config schema version

## Approach

Mirror the shipped `state-schema-version` pattern exactly, on the config plane.

- `config.Config` gains `SchemaVersion int \`toml:"schema_version,omitempty"\``.
- `const CurrentConfigSchemaVersion = 1`.
- In `config.Load`, immediately after `toml.Unmarshal`, add:
  ```go
  if c.SchemaVersion > CurrentConfigSchemaVersion {
      return nil, fmt.Errorf("parse config: unknown config schema version %d (this binary supports up to %d) — upgrade homonto", c.SchemaVersion, CurrentConfigSchemaVersion)
  }
  ```
  before the agents fold and the rest of processing.

Absent/0 → legacy → treated as current (backward-compatible; every existing
config loads unchanged). homonto never writes `homonto.toml`, so nothing stamps
the field — it is authored by users / future tooling, exactly as an int TOML key.

## Identity / safety

- No behavior change for any config without a future `schema_version`.
- The guard runs before any adapter/plan/apply, so a newer config is refused
  before it can be partially mis-applied.

## Test

TDD: `Load` of a config with `schema_version = <current+1>` errors with "upgrade
homonto"; a config omitting it, and one at the current version, load fine.

## Alternatives
- Config migrations pipeline (F43) — out of scope; this is only the version +
  forward-safety guard, the smallest useful X3/F37 slice.
