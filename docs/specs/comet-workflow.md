# comet-workflow Specification

## Purpose

Defines Homonto's current development workflow: Comet coordinates OpenSpec WHAT
artifacts with Superpowers HOW artifacts, state, verification, and archive.

## Requirements

### Requirement: Comet is the development entry point

New Homonto development SHALL start through `/comet` or a Comet preset. Agents
SHALL inspect `openspec/changes/` and each active change's `.comet.yaml` before
starting or resuming work. Agents SHALL NOT create new active `docs/changes/*`
Onto workspaces for Homonto development.

#### Scenario: No active change

- **GIVEN** `openspec list --json --no-color` returns no active changes
- **WHEN** the user requests new development work
- **THEN** the agent routes through `/comet-open` to create an OpenSpec change

### Requirement: OpenSpec is canonical for WHAT

New requirement changes SHALL be represented as OpenSpec changes under
`openspec/changes/<name>/`, with main specs under `openspec/specs/` after archive.
Existing `docs/specs/*.md` remain readable transition documents until a separate
conversion change migrates them.

#### Scenario: New capability

- **GIVEN** a new capability request
- **WHEN** Comet opens the change
- **THEN** proposal/design/tasks and any delta specs are created under
  `openspec/changes/<name>/`

### Requirement: Superpowers remains canonical for HOW

Deep technical design docs SHALL live under `docs/superpowers/specs/`, plans
under `docs/superpowers/plans/`, and verification reports under
`docs/superpowers/reports/`.

#### Scenario: Build phase planning

- **GIVEN** a Comet change in build phase
- **WHEN** the implementation plan is written
- **THEN** it is saved under `docs/superpowers/plans/` and its frontmatter links
  back to the OpenSpec change

### Requirement: Onto artifacts are legacy for development

`docs/changes/` SHALL be treated as legacy Onto history for Homonto development.
Archived workspaces MAY be consulted for historical context but SHALL NOT be
edited or used as active workflow state.

#### Scenario: Archived Onto change

- **GIVEN** an archived workspace under `docs/changes/archive/`
- **WHEN** an agent needs historical context
- **THEN** it may read the archive but must use current living docs and OpenSpec
  state for new work
