# codex-adapter Specification

## Purpose
TBD - created by archiving change adapter-contract-codex-pilot. Update Purpose after archive.
## Requirements
### Requirement: Codex MCP projection

homonto SHALL project declared MCP servers targeting Codex into
`~/.codex/config.toml` as `mcp_servers.<name>` tables holding command, args, and
env, surgically and idempotently, built on the shared projection core and a TOML
codec. Unmanaged tables and keys in `config.toml` SHALL be preserved, and a
consecutive plan SHALL report no changes.

#### Scenario: MCP server projected to config.toml

- **GIVEN** an MCP server declared with a codex target against a config.toml holding an unrelated key
- **WHEN** apply runs
- **THEN** `mcp_servers.<name>` holds the command/args/env and the unrelated key is preserved

#### Scenario: De-declared Codex MCP server is pruned

- **GIVEN** a Codex `mcp_servers.<name>` previously written and recorded by homonto, no longer declared
- **WHEN** apply runs
- **THEN** that server table is removed and any unmanaged server table is preserved

#### Scenario: Codex projection is idempotent

- **GIVEN** a Codex MCP server already applied
- **WHEN** plan runs twice consecutively
- **THEN** both plans report no changes

### Requirement: Codex secret safety

Codex projection SHALL preserve the secret guarantees: plan SHALL NOT resolve or
print a secret value (it shows the token), and apply SHALL resolve secrets only
after confirmation, storing only the unresolved token plus a hash in state.

#### Scenario: Plan does not reveal a Codex secret

- **GIVEN** a Codex MCP server whose env references a secret token
- **WHEN** plan runs
- **THEN** the plan shows the token or a redaction, never the resolved secret value

### Requirement: codex is covered by the shared adapter conformance suite

The codex adapter SHALL be exercised by the shared adapter conformance suite for
the surface it supports (MCP projection). Conformance checks that do not apply to
codex's reduced surface SHALL be explicitly skipped with a documented reason,
never silently omitted, so codex's supported-vs-unsupported surface is pinned by
the same suite that covers the other adapters.

#### Scenario: codex runs the applicable conformance checks

- **WHEN** the conformance suite runs against the codex adapter
- **THEN** the checks its MCP surface supports pass, and any inapplicable check is explicitly skipped with a reason
