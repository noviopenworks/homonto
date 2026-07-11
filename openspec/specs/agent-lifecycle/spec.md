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
