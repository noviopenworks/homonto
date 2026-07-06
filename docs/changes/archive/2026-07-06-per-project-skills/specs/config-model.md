# Delta Spec: config-model (per-project-skills)

## ADDED Requirements

### Requirement: Skill install scope

The config model SHALL expose an optional `[skills] scope` selecting where owned skills
install: `"user"` (the default when absent or empty) targets the user's home tool
directories, and `"project"` targets the project root (the directory of `homonto.toml`).
`config.Load` SHALL reject any other value, naming the offending value and the valid set
(`user`, `project`). Scope SHALL govern skill symlinks only; MCP servers and settings are
unaffected.

#### Scenario: Absent scope defaults to user
- **WHEN** `homonto.toml` has `[skills] own` but no `scope`
- **THEN** the loaded model's skill scope is `user`

#### Scenario: Project scope parsed
- **WHEN** `homonto.toml` sets `[skills] scope = "project"`
- **THEN** the loaded model's skill scope is `project`

#### Scenario: Invalid scope is rejected
- **GIVEN** `[skills] scope = "global"` (not a valid value)
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming `"global"` and the valid values `user` and
  `project`, rather than silently defaulting
