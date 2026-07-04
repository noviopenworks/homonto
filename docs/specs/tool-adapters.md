# tool-adapters Specification

## Purpose
TBD - created by archiving change homonto-v1-core. Update Purpose after archive.
## Requirements
### Requirement: Surgical merge preserves unmanaged keys

Each adapter SHALL write only the keys homonto manages and SHALL preserve all
unmanaged keys already present in a tool's file. A tool file that cannot be parsed
SHALL cause that adapter to abort and report, never to overwrite.

#### Scenario: Unmanaged keys survive apply
- **WHEN** a tool file contains keys homonto does not manage
- **THEN** those keys are byte-preserved (values intact) after apply

#### Scenario: Unparseable file is not clobbered
- **WHEN** an existing tool file cannot be parsed
- **THEN** that adapter aborts and reports and does not write the file, while
  other tools still proceed

### Requirement: Claude Code projection

The Claude adapter SHALL project MCP servers into `~/.claude.json`
(`mcpServers.<name>`), settings and plugins into `~/.claude/settings.json`, and
owned skills as symlinks under `~/.claude/skills/`.

#### Scenario: MCP and setting projected surgically
- **WHEN** apply runs with an MCP targeting claude and a claude setting
- **THEN** `mcpServers.<name>` is written to `~/.claude.json` and the setting to
  `~/.claude/settings.json`, with pre-existing unmanaged keys in both files intact

### Requirement: OpenCode projection

The OpenCode adapter SHALL project MCP servers into `opencode.jsonc`
(`mcp.<name>` with `type:"local"`, `command`, `enabled`, and `environment` when
env is set), settings as top-level keys, plugins appended to the `plugin` array,
and owned skills as symlinks under `~/.config/opencode/skills/`. JSONC input SHALL
be normalized before editing; loss of inline comments in rewritten regions is a
documented caveat.

#### Scenario: MCP projected with local shape and plugin appended
- **WHEN** apply runs with an MCP targeting opencode and an opencode plugin
- **THEN** `mcp.<name>.type` is `local` with the command, and the plugin is
  appended to the existing `plugin` array without duplicating existing entries

#### Scenario: Existing JSONC keys preserved
- **WHEN** `opencode.jsonc` has an unmanaged key and a comment
- **THEN** the unmanaged key survives after apply

### Requirement: Owned content linked by symlink with conflict detection

Owned skills SHALL be linked (not copied) from `content/skills/<name>` into each
tool's skills directory. If the target already exists and is not homonto's link,
the adapter SHALL report a conflict and SHALL NOT clobber it.

#### Scenario: Idempotent link creation
- **WHEN** a skill symlink does not yet exist
- **THEN** apply creates it, and a second apply reports no change for that link

#### Scenario: Conflict is reported, not clobbered
- **WHEN** the link target exists as a real file or points elsewhere
- **THEN** apply reports a conflict and leaves the existing file untouched

