# tool-adapters (delta)

## ADDED Requirements

### Requirement: adapters pass a shared conformance suite

Every tool adapter SHALL pass a shared, reusable conformance suite that exercises
the `Adapter` contract uniformly rather than relying only on per-adapter ad-hoc
tests. The suite SHALL assert at least: a `Plan` on a fresh config yields create
changes; `Apply` writes them; a subsequent `ObserveHashes` reports every applied
key as clean (unchanged); a second `Plan` yields no changes (idempotent); and an
unmanaged file present in the target tree is preserved across `Apply`.

#### Scenario: claude and opencode pass the core conformance checks

- **WHEN** the conformance suite runs against the claude and opencode adapters
- **THEN** each satisfies the create/observe-clean/idempotent-replan/unmanaged-preservation checks
