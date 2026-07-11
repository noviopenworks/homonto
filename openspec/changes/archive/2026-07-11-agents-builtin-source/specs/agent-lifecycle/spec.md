## MODIFIED Requirements

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
