# adapter-contract

## ADDED Requirements

### Requirement: Built-in JSON adapters project structured documents through the shared core

The `claude` and `opencode` adapters SHALL project their structured-document
managed keys (JSON config documents) through the shared
`internal/adapter/structproj` core — `Project`, `Apply`, and `Observe` — via a
shared JSON `Codec` backed by `internal/jsonutil`, rather than each
re-implementing the diff/write/observe control flow. Each adapter maps its
managed keys to the document they live in and supplies a `pathFor` per
document. The structured-document projection behavior — create/update/adopt/
delete/noop diffing, managed-key-only writes preserving unmanaged content,
secret-safe `Old` redaction, and drift re-hashing — MUST be identical to the
prior bespoke implementation, as pinned by the shared conformance suite.

This requirement covers only structured-document projection; file-projection
surfaces (symlinked skills/commands/subagents, copy-mode subagents) are out of
its scope and remain adapter-owned.

#### Scenario: Claude routes settings and .claude.json keys through the core

- **WHEN** the `claude` adapter plans, applies, and observes its managed
  `setting.*`, `mcp.*`, `plugin.*`, `pluginconfig.*`, and `marketplace.*` keys
- **THEN** it does so through `structproj.Project` / `Apply` / `Observe` with a
  shared JSON codec, and the resulting changes, on-disk writes, and observed
  hashes are byte-for-byte identical to the prior implementation

#### Scenario: OpenCode routes opencode.json keys through the core

- **WHEN** the `opencode` adapter plans, applies, and observes its managed
  `mcp.*` and `setting.*` keys in `opencode.json`
- **THEN** it does so through the shared `structproj` core and the shared JSON
  codec, preserving unmanaged content and secret-safe redaction unchanged

#### Scenario: Shared JSON codec is used by both JSON adapters

- **WHEN** either JSON adapter projects a structured document
- **THEN** it uses the one shared JSON `Codec` (backed by `internal/jsonutil`),
  not a per-adapter reimplementation of the format primitives
