# Comet Design Handoff

- Change: config-schema-version
- Phase: design
- Mode: compact
- Context hash: 0d928090e09a86ebdd1581ef5d583250a24706b73d737a6a5a40bd1d785da36d

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/config-schema-version/proposal.md

- Source: openspec/changes/config-schema-version/proposal.md
- Lines: 1-39
- SHA256: cf9e2f7487d29c7e0541e6c9103a35eb3db1a2d83761f06093d2788abdade3d6

```md
# Explicit homonto.toml config schema version, rejected fail-closed when unknown

## Why

Roadmap X3 (F37 config half). `state.json` gained a `schemaVersion` that is
rejected fail-closed when it exceeds what the binary understands
(`state-schema-version`, archived). `homonto.toml` has **no** such version: an
older binary reading a newer config written by a future homonto silently ignores
fields it does not understand (TOML unmarshal drops unknown keys), applying a
partial, wrong projection to the user's tools rather than telling them to
upgrade. The config plane deserves the same forward-safety the state plane has.

## What Changes

- Add an optional top-level `schema_version` (int) to `homonto.toml` / the
  `config.Config` struct. Absent or `0` means a legacy (pre-versioning) config
  and is treated as the current version — fully backward-compatible.
- `config.Load` rejects a config whose `schema_version` exceeds the version this
  binary supports, fail-closed, with a clear "upgrade homonto" error — before
  any adapter or apply logic runs. Mirrors `state.Load`'s check exactly.
- Introduce `config.CurrentConfigSchemaVersion` (starting at `1`).

## Impact

- **Specs:** `config-model` gains a requirement that config carries an explicit
  schema version and a config from a newer schema is rejected fail-closed.
- **Behavior:** none for existing configs (absent version = current). The only
  new behavior is that a config declaring a future `schema_version` is refused
  with guidance to upgrade, instead of being silently mis-applied.
- **Risk:** low — an additive field + one fail-closed guard at load, mirroring
  the shipped state-plane pattern; covered by new load tests + the full suite.

## Non-goals

- Config migrations / a decode→migrate→normalize→validate→expand pipeline (F43)
  — a larger refactor; this only adds the version and the forward-safety guard.
- The ToolID capability registry (F33/F34).
- Writing `schema_version` into user configs (homonto does not rewrite
  `homonto.toml`); the field is authored by users / future tooling.

```

## openspec/changes/config-schema-version/design.md

- Source: openspec/changes/config-schema-version/design.md
- Lines: 1-34
- SHA256: d0483bc4042873bcce2d29556f58a455e06f9f8b89403b385a3fcb2fdb400c0f

```md
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

```

## openspec/changes/config-schema-version/tasks.md

- Source: openspec/changes/config-schema-version/tasks.md
- Lines: 1-10
- SHA256: 3be9b81c41fd7b36b3930a12b3ae8519ab1f5fd9c33052904c8636639825ff43

```md
# Tasks — config-schema-version

## 1. Versioned config with fail-closed load
- [ ] Add Config.SchemaVersion (`toml:"schema_version,omitempty"`) +
      CurrentConfigSchemaVersion=1; config.Load rejects a future version
      fail-closed (absent/0 = current). TDD: a future version errors; absent
      and current load fine.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/config-schema-version/specs/config-model/spec.md

- Source: openspec/changes/config-schema-version/specs/config-model/spec.md
- Lines: 1-27
- SHA256: d3865d74dbda757c2a5bd212f651c58f06175152f6ddeeca8f77ea69cbc2dbee

```md
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

```
