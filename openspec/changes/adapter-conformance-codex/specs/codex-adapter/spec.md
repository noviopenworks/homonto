# codex-adapter (delta)

## ADDED Requirements

### Requirement: codex is covered by the shared adapter conformance suite

The codex adapter SHALL be exercised by the shared adapter conformance suite for
the surface it supports (MCP projection). Conformance checks that do not apply to
codex's reduced surface SHALL be explicitly skipped with a documented reason,
never silently omitted, so codex's supported-vs-unsupported surface is pinned by
the same suite that covers the other adapters.

#### Scenario: codex runs the applicable conformance checks

- **WHEN** the conformance suite runs against the codex adapter
- **THEN** the checks its MCP surface supports pass, and any inapplicable check is explicitly skipped with a reason
