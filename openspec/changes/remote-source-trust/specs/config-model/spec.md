## ADDED Requirements

### Requirement: Remote source form with required digest

The config model SHALL accept a `remote:` URL source on any resource that
already accepts `builtin:` or `local:` sources, and SHALL require a sibling
`digest` field holding a sha256 pin. A `remote:` source without a valid sha256
digest SHALL be a load-time error. The `digest` field SHALL NOT affect
non-remote sources, preserving existing `builtin:` and `local:` behavior
unchanged.

#### Scenario: Remote source with valid digest parses

- **GIVEN** `[subagents.x]` with `source = "remote:https://h.test/x.tgz"` and `digest = "sha256:<64 hex>"`
- **WHEN** the config loads
- **THEN** the resource carries a remote source and the recorded pin

#### Scenario: Remote source without digest is rejected

- **GIVEN** a `remote:` source with no `digest`
- **WHEN** the config loads
- **THEN** loading fails with a clear missing-pin error

#### Scenario: Builtin and local sources are unaffected

- **GIVEN** existing `builtin:`/`local:` resources with no `digest`
- **WHEN** the config loads
- **THEN** they load exactly as before
