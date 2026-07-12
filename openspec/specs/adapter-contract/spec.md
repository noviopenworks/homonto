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
