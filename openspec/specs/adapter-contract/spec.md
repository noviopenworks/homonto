# adapter-contract Specification

## Purpose
TBD - created by archiving change adapter-contract-codex-pilot. Update Purpose after archive.
## Requirements
### Requirement: Format-agnostic managed-key projection core

homonto SHALL provide a projection core that owns the managed-key control flow
for a structured config file, parameterized by a format Codec so a new adapter
supplies only its file path, key mapping, and codec. The core SHALL produce the
same create, update, delete, noop, and adopt changes the built-in adapters
produce, write only managed keys while preserving unmanaged content, and
re-hash recorded keys for drift detection.

#### Scenario: Declared key projects as create then noop

- **GIVEN** a managed key declared for a tool whose file lacks it
- **WHEN** plan runs, then apply, then plan again
- **THEN** the first plan is a create, apply writes the key, and the second plan is a noop

#### Scenario: De-declared key is pruned

- **GIVEN** a managed key recorded in state but no longer declared
- **WHEN** plan runs
- **THEN** the core emits a delete for that key and apply removes only it

#### Scenario: Unmanaged content is preserved

- **GIVEN** a config file holding keys homonto does not manage
- **WHEN** apply writes a managed key
- **THEN** every unmanaged key is preserved byte-for-byte outside the managed change

### Requirement: Codec abstracts the file format

The projection core SHALL depend only on a Codec that can get, set, delete, and
canonicalize a value at a key path in a document, and normalize an empty
document to an object root. A JSON codec and a TOML codec SHALL both satisfy the
Codec so the same core drives JSON- and TOML-configured tools.

#### Scenario: JSON and TOML codecs drive the same core

- **GIVEN** the projection core and equivalent desired state
- **WHEN** it runs with a JSON codec against a JSON file and a TOML codec against a TOML file
- **THEN** both produce equivalent managed-key changes and preserve unmanaged content

### Requirement: Adapter compatibility fixture contract

A conforming adapter SHALL be validated by a real-config compatibility fixture
that proves surgical merge, idempotency, pruning, and conflict safety.

#### Scenario: Compatibility fixture passes

- **GIVEN** a real config file with unmanaged user content and a managed declaration
- **WHEN** the fixture suite runs apply, re-plan, and de-declare
- **THEN** the managed key is projected, unmanaged content survives, the re-plan is byte-identical, the de-declared key is pruned, and a non-homonto value is never clobbered

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
