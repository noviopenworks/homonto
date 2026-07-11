# agent-lifecycle Specification

## Purpose
TBD - created by archiving change agents-foundation. Update Purpose after archive.
## Requirements
### Requirement: homonto agents list reports declared agents

`homonto agents list` SHALL be a read-only command that loads the config
(honoring `--config`) and prints each declared `[agents.<name>]` agent, sorted by
name, showing its name, source, version (or an unpinned indicator), targets, and
mode. It SHALL perform no projection and no mutation. When no agents are declared
it SHALL say so. `homonto agents` with no subcommand SHALL show help.

#### Scenario: List declared agents

- **GIVEN** a config with two `[agents.<name>]` agents
- **WHEN** `homonto agents list` runs
- **THEN** it prints both agents sorted by name, each with source, version-or-unpinned, targets, and mode, and exits 0

#### Scenario: No agents declared

- **GIVEN** a config with no `[agents]` section
- **WHEN** `homonto agents list` runs
- **THEN** it reports that no agents are declared and exits 0

#### Scenario: agents list is read-only

- **WHEN** `homonto agents list` runs
- **THEN** it writes no files and mutates no tool config or state

### Requirement: homonto agents add installs a declared agent

`homonto agents add <name>` SHALL install a declared `[agents.<name>]` agent into
its target tools and record the installation in an agent lockfile at
`.homonto/agents-lock.json`. This increment supports `local:<x>` sources only;
`builtin:` and remote sources SHALL return a clear "not yet supported" error.

For a `local:<x>` agent the command SHALL resolve `homonto/agents/<x>.md`
(relative to the config directory), and for each target in the agent's targets
install it into that tool's agent directory as `<name>.md`: `copy` mode writes the
content, `link` mode symlinks the source. The command SHALL be:

- **conflict-safe**: if a destination already exists and is not a homonto-managed
  install of this agent (not recorded in the lockfile), it SHALL refuse and
  install nothing for that agent;
- **idempotent**: a target already installed with matching content SHALL be a
  no-op;
- **recorded**: on success the lockfile SHALL record the agent's source, version,
  mode, targets, and each target's installed path and content hash.

An undeclared agent name SHALL be an error. A missing local source file SHALL be
an error naming the expected path.

#### Scenario: Add a local copy-mode agent

- **GIVEN** `[agents.rev]` with `source = "local:rev"` and `mode = "copy"`, a `homonto/agents/rev.md`, and both tools targeted
- **WHEN** `homonto agents add rev` runs
- **THEN** `rev.md` is written into each tool's agent directory, the lockfile records the agent with each target's path and content hash, and the command reports the installs

#### Scenario: Add is idempotent

- **GIVEN** an already-installed agent with unchanged content
- **WHEN** `homonto agents add <name>` runs again
- **THEN** each target is a no-op and nothing is rewritten

#### Scenario: Add refuses to clobber an unmanaged file

- **GIVEN** a destination `<name>.md` that already exists and is not recorded in the lockfile
- **WHEN** `homonto agents add <name>` runs
- **THEN** it refuses naming the conflict and installs nothing for that agent

#### Scenario: builtin source is not yet supported

- **GIVEN** `[agents.x]` with `source = "builtin:x"`
- **WHEN** `homonto agents add x` runs
- **THEN** it returns a clear error that builtin sources are not yet supported

#### Scenario: undeclared agent is an error

- **WHEN** `homonto agents add nope` runs against a config with no `[agents.nope]`
- **THEN** it errors that the agent is not declared

### Requirement: homonto agents doctor reports agent health

`homonto agents doctor` SHALL be a read-only command that loads the config
(declared agents) and `.homonto/agents-lock.json` (installed agents) and reports
each drift as a finding. It SHALL write nothing. It SHALL check:

- a declared agent absent from the lockfile is **not installed**;
- a lockfile-recorded agent absent from the config is **orphaned**;
- a `local:` agent whose `homonto/agents/<source>.md` content hash differs from
  the recorded install hash (or whose source file is missing) has a **source
  drift**;
- a target the agent declares but has no lockfile install entry is a **target not
  installed**;
- a lockfile install entry for a target the agent no longer declares is a
  **target no longer declared**;
- a recorded install path that no longer exists on disk is **missing on disk**;
- a `copy`-mode install whose on-disk content hash differs from the recorded hash
  is **modified on disk**.

On a healthy workspace it SHALL print `healthy` and exit 0. When one or more
findings exist it SHALL print each finding and exit non-zero.

#### Scenario: healthy workspace

- **GIVEN** a config whose declared agents are all installed, undrifted, and unmodified per the lockfile and disk
- **WHEN** `homonto agents doctor` runs
- **THEN** it prints `healthy` and exits 0

#### Scenario: declared but not installed

- **GIVEN** a `[agents.<name>]` with no lockfile record
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the agent as not installed and exits non-zero

#### Scenario: orphaned install

- **GIVEN** a lockfile agent that is no longer declared in the config
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the agent as orphaned and exits non-zero

#### Scenario: source drift

- **GIVEN** an installed `local:` agent whose source file content changed since install
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the source drift and exits non-zero

#### Scenario: modified on disk

- **GIVEN** a copy-mode installed agent whose on-disk file content was edited
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the file as modified on disk and exits non-zero

#### Scenario: missing on disk

- **GIVEN** a recorded install whose file was deleted
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the file as missing on disk and exits non-zero

#### Scenario: read-only

- **WHEN** `homonto agents doctor` runs
- **THEN** it writes no files and mutates nothing

### Requirement: homonto agents update re-materializes an installed agent

`homonto agents update <name>` SHALL re-install an already-installed declared
agent from its current source and refresh `.homonto/agents-lock.json`. The agent
MUST be declared and recorded in the lockfile; an undeclared or not-yet-installed
agent SHALL be an error (the latter directing the user to `agents add`). This
increment supports `local:<x>` sources only; `builtin:`/remote sources SHALL
return a clear "not yet supported" error.

For each of the agent's declared targets the command SHALL re-materialize per the
agent's mode: `copy` writes the current `homonto/agents/<x>.md` content; `link`
ensures the symlink points at the source. It SHALL be:

- **backup-preserving**: before overwriting a `copy`-mode target whose on-disk
  content differs from the recorded hash (a local edit), the current file SHALL be
  copied to `<path>.bak`;
- **idempotent**: a target already matching the source SHALL be a no-op;
- **recorded**: the lockfile SHALL be refreshed with each target's new content
  hash.

#### Scenario: update re-materializes a changed source

- **GIVEN** an installed copy-mode `local:` agent whose source file content changed since install
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target file is rewritten to the new source content and the lockfile hash is refreshed

#### Scenario: update backs up a locally-modified install

- **GIVEN** an installed copy-mode agent whose on-disk file was edited (differs from the recorded hash)
- **WHEN** `homonto agents update <name>` runs
- **THEN** the current file is copied to `<path>.bak` before the source content overwrites it

#### Scenario: update is idempotent

- **GIVEN** an installed agent already matching its source
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target is a no-op and no `.bak` is created

#### Scenario: update requires a prior install

- **GIVEN** a declared agent with no lockfile record
- **WHEN** `homonto agents update <name>` runs
- **THEN** it errors that the agent is not installed and points to `agents add`

#### Scenario: builtin source is not yet supported

- **GIVEN** an installed-or-declared `builtin:` agent
- **WHEN** `homonto agents update <name>` runs
- **THEN** it returns a clear error that builtin sources are not yet supported

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
