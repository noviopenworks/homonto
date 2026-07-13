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
