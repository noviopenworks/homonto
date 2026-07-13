# tool-adapters (delta)

## MODIFIED Requirements

### Requirement: adapters pass a shared conformance suite

Every tool adapter SHALL pass a shared, reusable conformance suite exercising the
`Adapter` contract uniformly. The suite SHALL assert at least: `Plan` on a fresh
config yields creates; `Apply` writes them; `ObserveHashes` reports applied keys
clean; a second `Plan` is a no-op; an unmanaged file is preserved across `Apply`;
a managed file changed out-of-band is reported as drift and reset by re-`Apply`; a
pre-existing malformed tool document does not crash `Plan`/`Apply`; a secret
reference in config is never resolved into a hash or leaked as plaintext through
`ObserveHashes`; and foreign on-disk content for an unowned key is not silently
clobbered or adopted outside the normal plan.

#### Scenario: claude and opencode pass the full conformance suite

- **WHEN** the conformance suite runs against the claude and opencode adapters
- **THEN** each satisfies the core, drift, malformed-doc, secret-non-resolution, and foreign-content-safety checks
