# cli-commands (delta)

## MODIFIED Requirements

### Requirement: Command surface

`homonto` SHALL expose the top-level commands `version`, `init`, `import`,
`plan`, `apply`, `status`, and `doctor`, with a persistent `--config` flag
(default `homonto.toml`). MCP servers, settings, plugins, marketplaces, TUI
settings, skills, commands, subagents, and frameworks are reconciled
declaratively through the `plan`/`apply` model by editing `homonto.toml`. The
deprecated `[agents.<name>]` table is also handled declaratively: it is folded
into an equivalent copy-mode subagent at config load and projected by `apply`
like any other subagent. There is no imperative `agents` command group.

#### Scenario: Version prints the build version
- **WHEN** the user runs `homonto version`
- **THEN** it prints `homonto <version>`

#### Scenario: Only declarative commands are registered
- **WHEN** the user runs `homonto --help`
- **THEN** it lists exactly `version`, `init`, `import`, `plan`, `apply`,
  `status`, and `doctor`
- **AND** no `agents` command group is present
