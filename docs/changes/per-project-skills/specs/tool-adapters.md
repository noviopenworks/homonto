# Delta Spec: tool-adapters (per-project-skills)

## MODIFIED Requirements

### Requirement: Claude Code projection

The Claude adapter SHALL project MCP servers into `~/.claude.json`
(`mcpServers.<name>`) and settings and plugins into `~/.claude/settings.json` — always at
the user's home, independent of skill scope. It SHALL link owned skills as symlinks under a
skills directory selected by the config's skill scope: `~/.claude/skills/` for `user` scope
and `<project>/.claude/skills/` for `project` scope, where `<project>` is the directory of
`homonto.toml`.

#### Scenario: MCP and setting projected surgically
- **WHEN** apply runs with an MCP targeting claude and a claude setting
- **THEN** `mcpServers.<name>` is written to `~/.claude.json` and the setting to
  `~/.claude/settings.json`, with pre-existing unmanaged keys in both files intact

#### Scenario: Project scope links skills under the project root
- **GIVEN** a config with `[skills] scope = "project"` owning a skill
- **WHEN** apply runs
- **THEN** the skill symlink is created under `<project>/.claude/skills/<name>` and nothing
  is added under `~/.claude/skills/`, while `~/.claude.json` and `~/.claude/settings.json`
  remain the projection targets for MCPs and settings

### Requirement: OpenCode projection

The OpenCode adapter SHALL project MCP servers into `opencode.jsonc`
(`mcp.<name>` with `type:"local"`, `command`, `enabled`, and `environment` when
env is set), settings as top-level keys, and plugins appended to the `plugin` array —
always at the user's home, independent of skill scope. It SHALL link owned skills as
symlinks under a skills directory selected by the config's skill scope:
`~/.config/opencode/skills/` for `user` scope and `<project>/.opencode/skills/` for
`project` scope, where `<project>` is the directory of `homonto.toml`. JSONC input SHALL be
normalized before editing; when homonto writes `opencode.jsonc`, all comments in that file
are removed by whole-document JSONC standardization.

#### Scenario: MCP projected with local shape and plugin appended
- **WHEN** apply runs with an MCP targeting opencode and an opencode plugin
- **THEN** `mcp.<name>.type` is `local` with the command, and the plugin is
  appended to the existing `plugin` array without duplicating existing entries

#### Scenario: Project scope links skills under the project root
- **GIVEN** a config with `[skills] scope = "project"` owning a skill
- **WHEN** apply runs
- **THEN** the skill symlink is created under `<project>/.opencode/skills/<name>` and
  nothing is added under `~/.config/opencode/skills/`

## ADDED Requirements

### Requirement: Skill scope relocation leaves no orphan

When a skill's install location changes because `[skills] scope` was switched, each adapter
SHALL relocate the skill's link rather than orphan the old one: `plan` SHALL render the move
as a single relocate change for `skill.<name>` (old location → new location), and `apply`
SHALL remove the managed symlink at the now-inactive scope location and create it at the
active one. The inactive-location removal SHALL follow the same conflict-safe rule as
pruning — only a symlink pointing into homonto's managed content directory is removed, an
absent path is a no-op, and a real file or foreign link is reported as a conflict and left
untouched. `user`-scope behavior with no scope change SHALL be identical to before this
capability.

#### Scenario: Switching scope relocates the link
- **GIVEN** a skill applied under `user` scope (linked at the home location) whose config is
  then changed to `[skills] scope = "project"`
- **WHEN** the user runs plan and confirms apply
- **THEN** plan shows a relocate for `skill.<name>` from the home location to
  `<project>/.claude/skills/<name>` (and the OpenCode equivalent), apply creates the
  project-location link and removes the home-location link, and a second plan reports no
  change

#### Scenario: Relocation prune is conflict-safe
- **GIVEN** a scope switch where the inactive-scope path holds a real file (not homonto's
  symlink)
- **WHEN** apply processes the relocation
- **THEN** the real file is left untouched and a conflict is reported, exactly as ordinary
  pruning behaves
