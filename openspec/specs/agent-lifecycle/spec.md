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

`homonto agents doctor` SHALL remain a read-only command reporting declared-vs-
installed drift with a non-zero exit on any problem finding. In the three-way-
merge model a locally-edited install (on-disk content differing from the recorded
base) is a normal, mergeable state and SHALL NOT be a problem finding. Doctor
SHALL still report: a declared-but-not-installed agent; an orphaned lockfile
agent; a `local:` source whose content differs from the recorded base ("source
changed"); a target declared-but-not-installed or installed-but-no-longer-
declared; a missing-on-disk install; and, newly, a **pending conflict** when a
`<dst>.merged` sidecar exists.

#### Scenario: locally-modified install is not a problem

- **GIVEN** an installed agent whose on-disk file was edited but whose source is unchanged
- **WHEN** `homonto agents doctor` runs
- **THEN** it does not report a problem for the local edit and (absent other issues) exits 0

#### Scenario: a pending merge conflict is reported

- **GIVEN** a `<dst>.merged` sidecar left by a conflicted `agents update`
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the target as conflicted (pointing at `<dst>.merged`) and exits non-zero

### Requirement: homonto agents update re-materializes an installed agent

`homonto agents update <name>` SHALL reconcile an already-installed declared
`local:` agent with its current source. The agent MUST be declared and recorded
in the lockfile; an undeclared or not-yet-installed agent SHALL be an error (the
latter directing the user to `agents add`). `builtin:`/remote sources SHALL return
a clear "not yet supported" error.

For each declared target in `copy` mode, with `BASE` = the recorded base content
(from the blob store), `LOCAL` = the on-disk file, and `UPSTREAM` = the current
source, the command SHALL:

- no-op when the on-disk content already equals the source ("up to date");
- when the base content is unavailable (no blob recorded, or the on-disk file is
  missing), fall back to backup-before-overwrite (a genuine local edit is copied
  to `<dst>.bak` before the source is written);
- otherwise perform a three-way merge (`merge.Merge(BASE, LOCAL, UPSTREAM)`):
  - **0 conflicts** → write the merged result to `<dst>` (backing up the prior
    local to `<dst>.bak` when it changes), and advance the recorded base to
    `UPSTREAM` (so the next update merges against the pristine source);
  - **≥1 conflict** → leave the live `<dst>` unchanged, write the
    merged-with-markers result to `<dst>.merged`, make no lockfile change, report
    the conflict, and exit non-zero.

`link`-mode targets are re-pointed only (no merge). The command SHALL remain
idempotent for an already-reconciled agent.

#### Scenario: non-overlapping local + upstream edits auto-merge

- **GIVEN** an installed copy agent, a local edit to one region, and a source edit to a disjoint region
- **WHEN** `homonto agents update <name>` runs
- **THEN** `<dst>` contains both edits, no `<dst>.merged` is created, and the recorded base advances to the source

#### Scenario: overlapping edits conflict via a sidecar

- **GIVEN** an installed copy agent whose local edit and source edit overlap
- **WHEN** `homonto agents update <name>` runs
- **THEN** the live `<dst>` is unchanged, a `<dst>.merged` with conflict markers is written, the lockfile is unchanged, and the command exits non-zero

#### Scenario: update is idempotent

- **GIVEN** an installed agent already equal to its source
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target is a no-op and no `.merged`/`.bak` is created

#### Scenario: missing base blob falls back to backup

- **GIVEN** an installed copy agent with a local edit but no recorded base blob
- **WHEN** `homonto agents update <name>` runs
- **THEN** the prior local is backed up to `<dst>.bak` and the source overwrites `<dst>`

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
