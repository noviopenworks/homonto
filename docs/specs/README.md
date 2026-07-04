# Living Capability Specs

One file per capability: `docs/specs/<capability>.md`. Each spec describes
what the system does **now** — always true, never a change log.

## Format

- `## Requirements`, containing one or more `### Requirement: <name>` blocks.
- Each requirement states a single SHALL sentence.
- Each requirement has one or more `#### Scenario: <name>` blocks written as
  **GIVEN / WHEN / THEN** bullets. Scenarios are the units the onto verify
  phase checks with fresh evidence.

## Lifecycle

- Living specs change only by merging a change's **delta spec**
  (`docs/changes/<name>/specs/<capability>.md`), which uses
  `## ADDED Requirements`, `## MODIFIED Requirements`,
  `## REMOVED Requirements`, and `## RENAMED Requirements` sections
  (canonical template: `onto-design/references/delta-spec.md`).
- `onto-close` lints deltas, then merges when a change is archived: ADDED
  blocks are appended, MODIFIED blocks replace the requirement of the same
  name, REMOVED blocks are deleted, RENAMED entries (`- FROM:` / `  TO:`
  pairs) rename the requirement heading preserving its body unless a
  MODIFIED block also targets the new name. A delta for a new capability
  creates the spec file.
