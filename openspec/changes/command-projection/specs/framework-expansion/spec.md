## MODIFIED Requirements

### Requirement: Framework metadata format

Each framework in the catalog SHALL have a `framework.toml` metadata file declaring `name`, `version`, `description`, optional `[dependencies] frameworks` list, and resource lists by kind (`[skills]` and `[commands]`, and later `[subagents]`). Each resource entry SHALL map a resource name to a catalog-relative path (`skills/<name>` for a skill directory, `commands/<name>.md` for a command file).

#### Scenario: Parse framework metadata

- **GIVEN** a framework `catalog/frameworks/comet/framework.toml` with name, version, dependencies, and a skills table
- **WHEN** Homonto loads the framework
- **THEN** it exposes the framework name, version, dependency names, and a map of skill names to catalog paths

#### Scenario: Parse framework command table

- **GIVEN** a framework `framework.toml` declaring a `[commands]` table mapping `demo-cmd = "commands/demo-cmd.md"`
- **WHEN** Homonto loads the framework
- **THEN** it exposes a map of command names to catalog command-file paths alongside the skills map
