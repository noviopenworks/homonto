# tool-adapters (delta)

## MODIFIED Requirements

### Requirement: adapters pass a shared conformance suite

Every tool adapter SHALL pass a shared, reusable conformance suite that exercises
the `Adapter` contract uniformly. The suite SHALL assert at least: a `Plan` on a
fresh config yields create changes; `Apply` writes them; a subsequent
`ObserveHashes` reports every applied key clean; a second `Plan` yields no changes
(idempotent); an unmanaged file present in the target tree is preserved across
`Apply`; a managed file changed out-of-band is reported by `ObserveHashes` as
differing from its recorded `Entry.Applied` and is reset by a re-`Apply`; and a
pre-existing malformed tool document does not crash `Plan` or `Apply` (it errors
or recovers, never panics).

#### Scenario: claude and opencode pass the extended conformance checks

- **WHEN** the conformance suite runs against the claude and opencode adapters
- **THEN** each satisfies the core checks plus drift-detection/reset and malformed-doc safety
