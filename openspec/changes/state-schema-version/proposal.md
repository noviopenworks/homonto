## Why
ROADMAP X3 / finding F37 (state half): homonto's `state.json` has no schema
version, so a future format change has no ordered/idempotent migration anchor and
an older binary reading a newer file would misinterpret it silently. Add a
`schemaVersion` to state (mirroring what onto's state got in N1): stamp it on
write, tolerate its absence (legacy), and reject a FUTURE version fail-closed.
## What Changes
- `state.State` gains `SchemaVersion int` (`schemaVersion,omitempty`); `Save`
  stamps `CurrentStateSchemaVersion` (=1); `Load` rejects a state whose
  `schemaVersion` exceeds the current (unknown/future), and treats absent/0 as
  the current legacy version.
## Impact
- **Code:** `internal/state/state.go` + test.
- **Spec:** `config-model` delta (state carries a versioned schema, future rejected).
- **Out of scope:** the config schema version + a full migration framework (F37 remainder), X1/X2.
