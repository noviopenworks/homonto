# framework-expansion Specification

## Purpose
TBD - created by archiving change catalog-foundation-skills. Update Purpose after archive.
## Requirements
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

### Requirement: Framework expansion from builtin source

When config declares `[frameworks.<name>] source = "builtin:<framework-name>"`, Homonto SHALL expand the framework into its constituent skill resources, each with an effective `source = "builtin:<skill-name>"`. Expansion SHALL include transitive dependencies: all dependency frameworks SHALL also be expanded, and their skills added to the effective resource set. Each expanded skill SHALL inherit the `[frameworks.<name>]` declaration's `scope` and `targets`, so a framework governs where its skills link and which tools receive them.

#### Scenario: Framework expands to its skills

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` where comet declares 8 skills
- **WHEN** the config is loaded
- **THEN** the effective skill set includes all 8 comet skills as builtin-source resources

#### Scenario: Expanded skills inherit framework scope and targets

- **GIVEN** `[frameworks.comet] source = "builtin:comet" scope = "user" targets = ["claude"]`
- **WHEN** the config is loaded and the framework is expanded
- **THEN** every expanded comet skill (and its transitive-dependency skills) carries `scope = "user"` and `targets = ["claude"]`

#### Scenario: Transitive dependency expansion

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` where comet depends on `superpowers` and `openspec`
- **WHEN** the config is loaded
- **THEN** the effective skill set includes comet's skills plus all skills from superpowers and openspec frameworks, deduplicated by name

### Requirement: Framework atomicity

A framework SHALL be enabled or disabled as an atomic unit. Individual framework-internal resources SHALL NOT be overridden or removed independently in this change. A loose builtin skill declared explicitly in `[skills.X]` with the same name as a framework skill SHALL produce a config error (name collision).

#### Scenario: Name collision between framework and loose skill

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` which includes skill `comet-open`, and also `[skills.comet-open] source = "builtin:comet-open"`
- **WHEN** the config is loaded
- **THEN** `config.Load` returns an error naming the collision

### Requirement: Dependency cycle detection

Framework dependency expansion SHALL detect cycles. If framework A depends on B and B depends on A (directly or transitively), `config.Load` SHALL return an error naming the cycle.

#### Scenario: Circular dependency rejected

- **GIVEN** framework A depends on B and B depends on A
- **WHEN** the config declares `[frameworks.A] source = "builtin:A"`
- **THEN** `config.Load` returns an error naming the circular dependency chain

### Requirement: First-release catalog frameworks

The catalog SHALL contain the four first-release frameworks: `onto`, `comet`, `superpowers`, and `openspec`. The `comet` framework SHALL declare dependencies on `superpowers` and `openspec`. The `onto` and `superpowers` and `openspec` frameworks SHALL have no dependencies.

#### Scenario: Comet framework declares correct dependencies

- **GIVEN** the bundled catalog
- **WHEN** the comet framework metadata is loaded
- **THEN** its dependencies list is `["superpowers", "openspec"]`

