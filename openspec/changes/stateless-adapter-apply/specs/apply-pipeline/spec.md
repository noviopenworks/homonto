# apply-pipeline

## ADDED Requirements

### Requirement: Applying a plan derives managed file entries from config

An adapter's apply step SHALL derive its managed file-projection entries
(skills, commands, subagents) from the configuration supplied to it, not from
mutable instance state left by a prior planning call. Apply MUST be correct when
given the same configuration the plan was computed from, without depending on a
prior plan call having populated the adapter instance. The resulting on-disk
links, files, and recorded state MUST be identical to deriving them during
planning.

#### Scenario: Apply is correct without relying on prior-plan instance state

- **WHEN** an adapter applies a change set with the configuration it was planned
  from
- **THEN** it derives its managed file entries from that configuration and
  produces the same links, files, and state as before — with no hidden
  dependence on instance fields set by a prior plan call
