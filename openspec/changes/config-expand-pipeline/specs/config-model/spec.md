# config-model

## ADDED Requirements

### Requirement: Framework resource expansion runs through one generic per-kind pipeline

Framework resource expansion SHALL run through a single generic pipeline
parameterized by the resource kind (skills, commands, subagents), rather than a
per-kind copy of the expansion logic. Every kind MUST expand,
tag (`builtin:<name>`), merge, and conflict-check identically — an
explicitly-declared resource also expanded by a framework, and a resource
expanded by two frameworks with conflicting scope/targets, MUST each fail with
the same rule for every kind, and the resulting expanded entries MUST be
identical to the prior per-kind implementation.

#### Scenario: Every kind expands through the same pipeline

- **WHEN** skills, commands, or subagents are expanded from framework declarations
- **THEN** the same expansion, tagging, merge, and conflict rules apply, producing
  the same entries as before
