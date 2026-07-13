# agent-lifecycle Specification

## Purpose
Records that the imperative agent lifecycle is retired. The former
`[agents.<name>]` model — an imperative agent command group, a per-agent
content-addressed base-blob store, three-way merge on upgrade, and a
lockfile-driven prune — no longer exists. `[agents.<name>]` is now a deprecated
backward-compatibility alias that folds into a subagent at config load (see the
`config-model` capability) and is projected declaratively by `apply` (see
`subagent-projection`). This capability is retained only as a tombstone so the
history and the non-existence guarantee are explicit.
## Requirements
### Requirement: Imperative agent lifecycle is retired

`homonto` SHALL NOT provide an imperative agent lifecycle: there SHALL be no
`agents` command group, no content-addressed base-blob store, and no
lockfile-driven prune. A `[agents.<name>]` declaration SHALL be folded into a
subagent at config load (see the `config-model` capability) and reconciled
declaratively by `apply`.

#### Scenario: No imperative agent command surface

- **WHEN** a user looks for an imperative agent install, update, doctor, or prune command
- **THEN** none is registered
- **AND** a declared `[agents.<name>]` is projected declaratively through `apply` as a subagent
