# apply-pipeline (delta)

## ADDED Requirements

### Requirement: a digest-only remote repin is shown and confirmed

`homonto` SHALL surface a digest-only remote repin as a change in `plan` and SHALL
require confirmation before `apply` mutates remote content for it. `homonto` SHALL
NOT apply a digest change under a "No changes / everything up to date" conclusion.

#### Scenario: a digest repin is not silently applied

- **GIVEN** a config whose only change is a remote source's pinned digest
- **WHEN** the user runs `plan` then `apply`
- **THEN** `plan` reports the digest change (not "no changes"), and `apply` mutates the remote content only after confirmation
