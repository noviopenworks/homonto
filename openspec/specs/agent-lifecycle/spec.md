# agent-lifecycle Specification

## Purpose
Defines the lifecycle-managed agent model that is distinct from the v1
projection-based subagent symlinks: declaring `[agents.<name>]` resources in
`homonto.toml`, the imperative `homonto agents` command group
(`list`/`add`/`update [--all]`/`doctor`/`prune`), conflict-safe installation of
`local:` and `builtin:` agent content into each tool's agent directory, the
per-agent content-addressed base-blob store that anchors three-way merging of a
user's local edits with a source/catalog upgrade, the pure line-based merge
engine, and the lockfile-driven prune of stale managed installs.
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
its target tools and record the installation in `.homonto/agents-lock.json`. The
agent's source SHALL be resolved as follows:

- `local:<x>` → `homonto/agents/<x>.md` (relative to the config directory);
- `builtin:<x>` → the embedded catalog's curated agent content by name (an
  unknown builtin name is an error);
- any other scheme (e.g. remote) → a clear "not yet supported" error.

For each target in the agent's targets it installs the resolved content into that
tool's agent directory as `<name>.md` (`copy` writes the content, `link` symlinks
a local source). The command SHALL be conflict-safe (refuse to clobber an
unmanaged file, all-or-nothing per agent), idempotent, and record each target's
path and content hash plus persist the base content to the blob store. An
undeclared agent name, or an unresolvable source, SHALL be an error.

#### Scenario: Add a builtin agent

- **GIVEN** a `[agents.rev]` with `source = "builtin:<name>"` where `<name>` is a curated catalog agent
- **WHEN** `homonto agents add rev` runs
- **THEN** the catalog content is installed into each target and recorded in the lockfile

#### Scenario: Add an unknown builtin agent is an error

- **GIVEN** a `[agents.x]` with `source = "builtin:not-a-real-agent"`
- **WHEN** `homonto agents add x` runs
- **THEN** it errors that the builtin agent is unknown

#### Scenario: Add a local copy-mode agent

- **GIVEN** `[agents.rev]` with `source = "local:rev"` and `mode = "copy"`, a `homonto/agents/rev.md`, and both tools targeted
- **WHEN** `homonto agents add rev` runs
- **THEN** `rev.md` is written into each tool's agent directory, the lockfile records the agent with each target's path and content hash, and the command reports the installs

#### Scenario: Add refuses to clobber an unmanaged file

- **GIVEN** a destination `<name>.md` that already exists and is not recorded in the lockfile
- **WHEN** `homonto agents add <name>` runs
- **THEN** it refuses naming the conflict and installs nothing for that agent

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

`homonto agents update <name>` (and `--all`) SHALL reconcile an already-installed
declared agent with its current source, resolving the source the same way as
`agents add` (`local:` from `homonto/agents/`, `builtin:` from the embedded
catalog, other schemes unsupported). The three-way merge, `.merged` conflict
sidecar, base-blob advance, backup fallback, and idempotency SHALL apply to
`builtin:` agents exactly as to `local:` — including auto-merging a user's local
edits with a catalog upgrade to a builtin agent. An undeclared, not-yet-installed,
or unresolvable-source agent SHALL be an error.

#### Scenario: update merges a catalog upgrade into a builtin agent's local edits

- **GIVEN** an installed `builtin:` copy agent with a local edit, and a newer catalog whose content for that agent changed disjointly
- **WHEN** `homonto agents update <name>` runs
- **THEN** the local edit and the catalog change are three-way-merged (or a `<dst>.merged` sidecar is written on conflict)

#### Scenario: update requires a prior install

- **GIVEN** a declared agent with no lockfile record
- **WHEN** `homonto agents update <name>` runs
- **THEN** it errors that the agent is not installed and points to `agents add`

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

### Requirement: homonto agents update --all reconciles every installed agent

`homonto agents update --all` SHALL run the three-way merge (the same as `homonto
agents update <name>`) across every agent recorded in `.homonto/agents-lock.json`,
and print a summary of the outcome (merged/updated, up-to-date, conflicted,
skipped). An agent still declared in the config SHALL be merged exactly as the
single-agent update would; an installed agent no longer declared in the config
SHALL be skipped with a note; a per-agent error (e.g. a missing local source)
SHALL be reported for that agent without aborting the rest. The command SHALL exit
non-zero if any agent had a conflict or a per-agent error, and exit 0 when all
agents are clean. `agents update` with neither a name nor `--all`, or with both,
SHALL be a usage error; single `agents update <name>` behavior is unchanged.

#### Scenario: update --all merges every installed agent

- **GIVEN** two installed agents, one with a disjoint local+source edit and one already up-to-date
- **WHEN** `homonto agents update --all` runs
- **THEN** the first is auto-merged and the second reported up-to-date, a summary is printed, and the command exits 0

#### Scenario: update --all exits non-zero on any conflict

- **GIVEN** two installed agents, one of which has an overlapping (conflicting) edit
- **WHEN** `homonto agents update --all` runs
- **THEN** the conflicting agent gets a `.merged` sidecar and the command exits non-zero, while the other agent is still processed

#### Scenario: update --all skips an orphaned agent

- **GIVEN** an installed agent that is no longer declared in the config
- **WHEN** `homonto agents update --all` runs
- **THEN** it is skipped with a note and does not cause the whole run to fail (absent other issues, exit 0)

#### Scenario: name and --all are mutually exclusive

- **WHEN** `homonto agents update <name> --all` runs (or `agents update` with neither)
- **THEN** it is a usage error

### Requirement: homonto agents prune removes stale managed installs

`homonto agents prune` SHALL remove homonto-managed agent installs that are no
longer declared and drop their lockfile records. It SHALL handle:

- an **orphan agent** (recorded in `.homonto/agents-lock.json` but not declared in
  the config): remove each recorded target install file and drop the agent's
  lockfile entry;
- a **de-declared target** (a target in an agent's `Installed` no longer in the
  agent's declared targets): remove that target's install file and drop its
  `Installed` entry, keeping the agent and its still-declared targets.

It SHALL only touch a file at a homonto-recorded install path. Before removing a
file whose on-disk content differs from the recorded base hash (a local edit), it
SHALL back the file up to `<path>.bak`. It SHALL also remove a pruned target's
leftover `<path>.merged` sidecar. It SHALL report each pruned item and print
`nothing to prune` when there is nothing to remove. A `--dry-run` flag SHALL list
what would be pruned and change nothing.

#### Scenario: prune an orphaned agent

- **GIVEN** an agent recorded in the lockfile that is no longer declared in the config
- **WHEN** `homonto agents prune` runs
- **THEN** its recorded install files are removed and its lockfile entry is dropped

#### Scenario: prune a de-declared target

- **GIVEN** an agent whose lockfile records a target the agent no longer declares
- **WHEN** `homonto agents prune` runs
- **THEN** that target's install file is removed and its `Installed` entry dropped, while the agent and its still-declared targets remain

#### Scenario: prune backs up a locally-modified install

- **GIVEN** an orphan agent whose install file was locally edited (differs from the recorded base hash)
- **WHEN** `homonto agents prune` runs
- **THEN** the file is copied to `<path>.bak` before being removed

#### Scenario: nothing to prune

- **GIVEN** a workspace whose lockfile matches the config exactly
- **WHEN** `homonto agents prune` runs
- **THEN** it reports nothing to prune and changes nothing

#### Scenario: dry run changes nothing

- **GIVEN** an orphan agent
- **WHEN** `homonto agents prune --dry-run` runs
- **THEN** it lists the orphan as prunable but removes no files and does not change the lockfile
