# command-projection Specification

## Purpose
TBD - created by archiving change command-projection. Update Purpose after archive.
## Requirements
### Requirement: Builtin and local command source resolution

A command resource SHALL resolve its content by source scheme: `[commands.<name>] source = "builtin:<name>"` resolves from the embedded catalog at `catalog/commands/<name>.md` (materialized to `.homonto/catalog/commands/<name>.md` on apply), and `source = "local:<name>"` resolves from `homonto/commands/<name>.md`. Commands are single Markdown files, not directories. Every command resource SHALL declare a `scope` (`user` or `project`) exactly as skills do.

#### Scenario: Builtin command resolves from materialized catalog

- **GIVEN** a config with `[commands.demo] source = "builtin:demo"` and `scope = "user"`
- **WHEN** apply runs
- **THEN** `catalog/commands/demo.md` is materialized to `.homonto/catalog/commands/demo.md` and the command link targets that file

#### Scenario: Local command resolves from homonto/commands

- **GIVEN** a config with `[commands.mine] source = "local:mine"` and `scope = "project"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `homonto/commands/mine.md`

### Requirement: Single-file command materialization

Homonto SHALL materialize builtin command content as single files from the
embedded catalog to `.homonto/catalog/commands/<name>.md` before creating command
symlinks, version-gated on the same catalog version tracked in state as skills.
Re-materialization SHALL occur only when the catalog version changes or the target
file is missing, and the catalog version SHALL be recorded only after a
successful materialization.

#### Scenario: First command materialization

- **GIVEN** no `.homonto/catalog/commands/demo.md` exists
- **WHEN** apply runs with a config declaring a builtin command `demo`
- **THEN** `.homonto/catalog/commands/demo.md` is written from the embedded catalog

#### Scenario: Version-gated command skip

- **GIVEN** `.homonto/catalog/commands/demo.md` exists and state records the current catalog version
- **WHEN** apply runs again with the same binary
- **THEN** the command is not re-materialized and the link is a no-op

### Requirement: Command projection into tool command directories

Owned commands SHALL be linked (not copied) into each tool's command directory at
the location chosen by the resource's `scope`: Claude at
`~/.claude/commands/<name>.md` (user) or `<repo>/.claude/commands/<name>.md`
(project), and OpenCode at `~/.config/opencode/command/<name>.md` (user) or
`<repo>/.opencode/command/<name>.md` (project). Pending link work SHALL appear as
plan changes (create / update / no-op). `apply` SHALL record each applied command
link in state and SHALL prune a de-declared command's link only when it is a
symlink pointing into a homonto-managed root (`homonto/commands/` or
`.homonto/catalog/commands/`); a real file or foreign link SHALL be reported as a
conflict and never clobbered.

#### Scenario: Builtin command links into both tools

- **GIVEN** a config with `[commands.demo] source = "builtin:demo"` targeting claude and opencode
- **WHEN** apply runs
- **THEN** `~/.claude/commands/demo.md` and `~/.config/opencode/command/demo.md` are symlinks into `.homonto/catalog/commands/demo.md`

#### Scenario: Idempotent command link

- **WHEN** a command link already points at its materialized target
- **THEN** plan reports no change and a second apply is a no-op

#### Scenario: Conflict is reported, not clobbered

- **GIVEN** a real file already exists at the command's link destination
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: De-declared command pruned only when it is our link

- **GIVEN** a command removed from `homonto.toml` whose link is a symlink into a homonto-managed root
- **WHEN** apply processes the delete
- **THEN** the link is removed; a real file or foreign link at that path is instead reported as a conflict and left untouched

### Requirement: Framework command expansion

A `framework.toml` `[commands]` table SHALL expand through `[frameworks.<name>] source = "builtin:<framework>"` into effective command resources with `source = "builtin:<command-name>"`, each inheriting the framework declaration's `scope` and `targets`, transitively across dependency frameworks and deduplicated by name, exactly as skills expand. A command name colliding with an explicit `[commands.X]` entry SHALL be a config error.

#### Scenario: Framework expands its commands

- **GIVEN** `[frameworks.demo] source = "builtin:demo"` where the demo framework declares one command `demo-cmd`
- **WHEN** the config is loaded
- **THEN** the effective command set includes `demo-cmd` as a builtin-source command inheriting the framework's scope and targets

### Requirement: Command link doctor verification

`doctor` SHALL verify each recorded command link: a builtin command's materialized
target under `.homonto/catalog/commands/` SHALL exist, and the tool-side symlink
SHALL be present and point at the expected source; a missing materialized file or
broken link SHALL be reported like a broken skill link.

#### Scenario: Doctor reports a linked command

- **GIVEN** a builtin command materialized and linked into a tool
- **WHEN** `doctor` runs
- **THEN** it reports the command link as present and correct

### Requirement: Placeholder fixture command

The first release of this capability SHALL ship exactly one placeholder command in
`catalog/commands/` so the machinery is materialized, linked, and dogfooded
end-to-end; real command content and framework-declared commands are populated by
a later change.

#### Scenario: Fixture command is projectable

- **GIVEN** the bundled catalog containing the placeholder command
- **WHEN** it is declared and applied
- **THEN** it materializes and links into both tools with no drift

