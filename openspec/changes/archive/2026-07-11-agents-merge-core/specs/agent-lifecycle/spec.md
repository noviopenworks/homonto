## ADDED Requirements

### Requirement: Three-way merge engine

The repository SHALL provide a pure, dependency-free line-based three-way merge:
`merge.Merge(base, local, upstream []byte) (result []byte, conflicts int)`. It
SHALL auto-merge changes that `local` and `upstream` make to disjoint regions of
`base`, and SHALL emit git-style conflict markers (`<<<<<<< local`, `=======`,
`>>>>>>> source`) for regions both sides changed differently, returning the count
of conflict regions. When a side is unchanged relative to base, the other side's
content SHALL be taken; when both sides made the identical change, it SHALL be
taken once.

#### Scenario: no changes

- **WHEN** `Merge(x, x, x)` is called
- **THEN** it returns `x` with 0 conflicts

#### Scenario: only local changed

- **GIVEN** `local` differs from `base` and `upstream == base`
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** it returns `local` with 0 conflicts

#### Scenario: only upstream changed

- **GIVEN** `upstream` differs from `base` and `local == base`
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** it returns `upstream` with 0 conflicts

#### Scenario: non-overlapping changes auto-merge

- **GIVEN** `local` edits an early region and `upstream` edits a later, disjoint region of `base`
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** it returns a result containing both edits with 0 conflicts

#### Scenario: overlapping changes conflict

- **GIVEN** `local` and `upstream` change the same region differently
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** the result contains a conflict-marked region and the conflict count is ≥ 1

### Requirement: Agent base-content blob store

Install operations SHALL persist installed agent content to a content-addressed
store `.homonto/agents-blobs/<sha256>`. `agentblob.Put(homontoDir, content)` SHALL
write the blob idempotently and return its sha256 hex (matching the lockfile
`Hash`); `agentblob.Get(homontoDir, hash)` SHALL read it back. `homonto agents
add` and `homonto agents update` SHALL `Put` each installed target's content, so
the base content is retrievable by the recorded install hash. This SHALL NOT
change the user-visible behavior of `add`/`update`.

#### Scenario: install persists a retrievable base blob

- **GIVEN** a local agent installed via `homonto agents add`
- **WHEN** the install completes
- **THEN** `.homonto/agents-blobs/<hash>` exists for each target's recorded hash and `agentblob.Get` returns the installed content

#### Scenario: blob Put is idempotent and content-addressed

- **WHEN** `agentblob.Put` is called twice with the same content
- **THEN** both return the same hash and the store holds a single blob
