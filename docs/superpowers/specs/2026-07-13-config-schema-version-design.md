---
comet_change: config-schema-version
role: technical-design
canonical_spec: openspec
status: draft
---

# config-schema-version — Technical Design

Deep design for X3's F37 config half. OpenSpec is the canonical spec; this
defers to `openspec/changes/config-schema-version/specs` for normative
requirements; the full approach is in the change's `design.md`.

## Decision

Give `homonto.toml` the same forward-safety the state file already has: add an
optional top-level `schema_version` and reject a config whose version exceeds
what the binary supports, fail-closed, before any adapter/plan/apply. Absent/0 =
legacy = current, so every existing config loads unchanged.

## Why

TOML unmarshal silently drops unknown keys, so today an older binary reading a
newer config mis-applies a partial projection instead of telling the user to
upgrade. `state.json` already guards this (`state-schema-version`, archived); the
config plane should match.

## Risk posture

Low — additive field + one guard at `config.Load` mirroring `state.Load`. No
behavior change for existing configs; covered by new load tests + the full suite.

## Out of scope

Config migrations / decode→migrate→normalize→validate→expand pipeline (F43); the
ToolID capability registry (F33/F34).
