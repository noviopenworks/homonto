## ADDED Requirements

### Requirement: Remote resolution routes through the trust pipeline

When the apply pipeline resolves a resource whose source is `remote:`, it SHALL
route resolution through the remote trust pipeline (cache lookup → verified
fetch → validate → pin-match → revocation) and materialize only from the
content-addressed cache. `builtin:` and `local:` resolution SHALL be unchanged.
A remote resolution failure SHALL abort the apply before any target mutation,
consistent with the atomic-writes / state-last guarantee.

#### Scenario: Remote resource projects like a managed resource

- **GIVEN** a pinned, cacheable `remote:` subagent/skill/command
- **WHEN** plan then apply runs
- **THEN** it materializes into each target tool exactly like a builtin/local resource, and status/doctor track it

#### Scenario: Remote resolution failure aborts apply cleanly

- **GIVEN** a `remote:` resource whose content fails verification
- **WHEN** apply runs
- **THEN** the apply aborts before any target file is written and existing state is unchanged

#### Scenario: Idempotent remote apply

- **GIVEN** an already-applied pinned remote resource
- **WHEN** apply runs again
- **THEN** it is a no-op (cache hit, no network, no target rewrite)
