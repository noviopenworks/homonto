# Delta spec — canonical template

One file per affected capability: `docs/changes/<name>/specs/<capability>.md`.
Deltas are living documents during build; onto-close lints them, then
merges into `docs/specs/<capability>.md`.

## Template

```markdown
# Delta Spec: <capability> (<change-name>)

## ADDED Requirements

### Requirement: <name>

<First line MUST contain SHALL or MUST.> <single-behavior statement>

#### Scenario: <name>

- **GIVEN** <precondition>
- **WHEN** <action>
- **THEN** <observable outcome>

## MODIFIED Requirements

### Requirement: <exact existing name>

<full replacement text — MODIFIED replaces the whole requirement,
scenarios included; first line MUST contain SHALL or MUST>

#### Scenario: <name>

- **GIVEN** … / **WHEN** … / **THEN** …

## REMOVED Requirements

### Requirement: <exact existing name>

<one line: why it no longer holds>

## RENAMED Requirements

- FROM: <exact existing name>
  TO: <new name>
```

## Rules (lint-enforced at close)

- Section headings: only `## ADDED|MODIFIED|REMOVED|RENAMED Requirements`;
  omit empty sections.
- Every requirement's **first line** contains SHALL or MUST.
- Every ADDED/MODIFIED requirement has ≥1 `#### Scenario:` with
  GIVEN/WHEN/THEN bullets — scenarios are what verify demands evidence
  for; an unverifiable requirement is a lint finding.
- MODIFIED/REMOVED/RENAMED names must match the living spec exactly.
- RENAMED preserves the body unless a MODIFIED block also targets the new
  name.
