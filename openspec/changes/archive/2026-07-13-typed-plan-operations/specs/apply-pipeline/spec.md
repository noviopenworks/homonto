# apply-pipeline

## ADDED Requirements

### Requirement: Apply validates every operation before any side effect

`Apply` SHALL validate every planned change set before performing any secret
resolution, remote or catalog materialization, or adapter write. A change set
whose tool is not a registered adapter, or that contains an operation whose
action is not one of the defined operations (create, update, delete, noop,
adopt), MUST abort the apply with an error naming the offending tool or action,
leaving no file or state mutated. Legal plans — every operation a registered
adapter emits — MUST be unaffected.

#### Scenario: Unknown tool aborts apply

- **WHEN** a change set names a tool that is not a registered adapter
- **THEN** apply aborts with an error and performs no write (the set is not
  silently skipped)

#### Scenario: Unknown action aborts apply

- **WHEN** a change set contains an operation whose action is not one of the
  defined operations
- **THEN** apply aborts with an error naming the offending action and performs
  no write

#### Scenario: Legal plan applies unchanged

- **WHEN** every change set carries a registered tool and only defined actions
- **THEN** validation passes and apply proceeds exactly as before
