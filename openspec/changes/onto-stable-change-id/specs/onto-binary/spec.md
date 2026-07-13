# onto-binary

## ADDED Requirements

### Requirement: A change carries a stable, name-independent id

`onto new` SHALL assign each change a stable identifier stored as `id` in its
`onto-state.yaml` — a content-independent value generated once at creation that
is never rewritten by any later command (`set`, `advance`, `close` preserve it
verbatim), so a change's identity survives a rename of its name or directory.
`onto state --json` and `onto status` MUST surface the id. A legacy state file
with no `id` MUST load with an empty id (backward compatible) and MUST NOT have
one retroactively minted, so an id never changes meaning across reads.

#### Scenario: onto new assigns a stable unique id

- **WHEN** two changes are created with `onto new`
- **THEN** each has a non-empty `id` in its `onto-state.yaml`, the two ids differ,
  and each id is unchanged by subsequent `advance`/`set`

#### Scenario: a legacy state without an id loads unchanged

- **WHEN** an `onto-state.yaml` written before this feature (no `id`) is read
- **THEN** it loads with an empty id and no id is minted on read
