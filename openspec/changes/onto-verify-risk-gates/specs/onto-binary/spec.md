# onto-binary

## ADDED Requirements

### Requirement: onto verification scale is risk-aware with non-waivable critical classes

The onto workflow's verification SHALL scale by risk as well as size and SHALL
treat a defined set of finding classes as non-waivable. A change whose diff
touches a security-sensitive surface (secret resolution, remote fetch or verify,
file deletion or pruning, or permission or ownership) MUST receive full
verification regardless of its file count, so a small security-relevant change is
never under-scrutinized. A finding of a security defect, data loss, or a failed
core-acceptance scenario MUST be treated as critical and fixed — it MUST NOT be
waived, skipped, or accepted as a deviation, in any verification mode. This is an
agent-judgment guarantee enforced by the onto-verify skill (per the B1 decision
the binary enforces the presence and shape of the verification result, not the
judgment behind it).

#### Scenario: A small security-relevant change gets full verification

- **WHEN** a change's diff touches a security-sensitive surface, even in a single file
- **THEN** the onto-verify skill directs full verification, not light

#### Scenario: A critical finding class cannot be waived

- **WHEN** verification surfaces a security defect, data loss, or a failed core-acceptance scenario
- **THEN** it must be fixed and cannot be recorded as a waived or skipped finding, in any mode
