# framework-expansion Specification

## Purpose
Defines the catalog framework metadata format (`framework.toml` with name,
version, dependencies, and resource tables for skills, commands, and subagents)
and the rules by which declaring `[frameworks.<name>] source = "builtin:<name>"`
expands into effective builtin skill/command/subagent resources that inherit the
framework declaration's `scope` and `targets`, transitively across dependency
frameworks and deduplicated by name, with atomic enablement, name-collision
rejection, dependency-cycle detection, and the first-release set of bundled
frameworks.
## Requirements
### Requirement: Framework metadata format

Each framework in the catalog SHALL have a `framework.toml` metadata file declaring `name`, `version`, `description`, optional `[dependencies] frameworks` list, and resource lists by kind (`[skills]`, `[commands]`, and `[subagents]`). Each resource entry SHALL map a resource name to a catalog-relative path (`skills/<name>` for a skill directory, `commands/<name>.md` for a command file, `subagents/<name>.md` for a subagent file).

#### Scenario: Parse framework metadata

- **GIVEN** a framework `catalog/frameworks/comet/framework.toml` with name, version, dependencies, and a skills table
- **WHEN** Homonto loads the framework
- **THEN** it exposes the framework name, version, dependency names, and a map of skill names to catalog paths

#### Scenario: Parse framework command table

- **GIVEN** a framework `framework.toml` declaring a `[commands]` table mapping `demo-cmd = "commands/demo-cmd.md"`
- **WHEN** Homonto loads the framework
- **THEN** it exposes a map of command names to catalog command-file paths alongside the skills map

#### Scenario: Parse framework subagent table

- **GIVEN** a framework `framework.toml` declaring a `[subagents]` table mapping `demo-agent = "subagents/demo-agent.md"`
- **WHEN** Homonto loads the framework
- **THEN** it exposes a map of subagent names to catalog subagent-file paths alongside the skills and commands maps

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

### Requirement: a non-builtin framework source fails at load

`homonto` SHALL reject at config load a `[frameworks.<name>]` declaration whose
source is not a `builtin:` source, with a clear error naming the framework and its
source. Only builtin frameworks are expanded; a `local:` or `remote:` framework
source would expand nothing, so it SHALL be a load error rather than a silent
no-op.

#### Scenario: a local framework source is rejected

- **GIVEN** a config with `[frameworks.onto] source = "local:onto"`
- **WHEN** the config is loaded
- **THEN** it is rejected naming the framework and the unsupported source, and nothing is installed

#### Scenario: a builtin framework source still loads

- **GIVEN** a config with `[frameworks.onto] source = "builtin:onto"`
- **WHEN** the config is loaded
- **THEN** it loads and the framework expands normally

### Requirement: The framework model supports versioned manifests and validated custom-source resolution

The framework ecosystem SHALL support versioned framework manifests and a single
validated resolution path that a builtin, a fourth builtin, or a trusted custom
framework all pass through. A framework manifest MAY declare a manifest schema
version, provided/required capabilities, and compatibility ranges; loading MUST
reject a manifest whose schema version exceeds what the binary supports (fail
closed), and MUST reject an incompatible framework or an unresolved required
capability with a clear error rather than silently installing nothing. The
existing guarantees — transitive dependency resolution, cycle detection, and
duplicate-resource rejection — MUST be preserved.

This requirement is recorded as the design target for roadmap E1; the design is
delivered and reviewed before implementation, which lands in later phased changes.

#### Scenario: A manifest from a newer schema is rejected

- **WHEN** a framework manifest declares a manifest schema version greater than
  the binary supports
- **THEN** loading fails closed with an "upgrade homonto" error and installs
  nothing

#### Scenario: A custom framework resolves through the same validated path

- **WHEN** a trusted custom framework is resolved
- **THEN** it is loaded and validated through the same manifest/dependency/
  path checks as a builtin framework, and an unsupported source or an
  incompatible version fails loudly

### Requirement: Framework dependency version ranges are validated fail-loud

Catalog loading SHALL validate every constrained framework dependency, where a
dependency of the form `"name@<constraint>"` compares the target framework's
three-part `x.y.z` version against `<constraint>` (`>=`, `>`, `<=`, `<`, `=`, or
a bare exact version). The dependency framework MUST exist and its version MUST
satisfy the constraint, otherwise loading fails with an error naming the
framework, the dependency, the version, and the constraint. A bare dependency
name (no constraint) MUST mean any version, preserving existing behavior, and the
dependency graph used for transitive resolution and cycle detection MUST continue
to key on the dependency name.

#### Scenario: An out-of-range dependency fails to load

- **WHEN** a framework declares a dependency `"dep@>=2.0.0"` and the indexed
  `dep` framework is version `1.0.0`
- **THEN** catalog loading fails with an error naming the framework, `dep`, the
  version, and the constraint

#### Scenario: A satisfied or bare dependency loads

- **WHEN** a dependency constraint is satisfied by the target's version, or the
  dependency is a bare name with no constraint
- **THEN** the catalog loads and transitive resolution behaves as before

### Requirement: The catalog can merge validated overlay framework sources

Catalog construction SHALL support merging one or more overlay framework sources
over a base source, validating every source through the same checks (manifest
schema, name-equals-directory, resource-path existence, dependency ranges). An
overlay that redefines a resource name already provided by an earlier source with
a different path MUST be rejected (strict conflict policy); an identical
name-to-path mapping collapses idempotently. Loading with no overlays MUST be
identical to loading the base source alone, and dependency-range validation MUST
run once after all sources are indexed so a cross-source dependency is checked.

#### Scenario: An overlay adds a framework over the base

- **WHEN** the catalog is loaded with a base source and an overlay providing a
  new framework
- **THEN** both frameworks and their resources are indexed and expandable

#### Scenario: An overlay may not shadow a base resource

- **WHEN** an overlay declares a resource name already provided by the base with
  a different path
- **THEN** loading fails with a conflict error

#### Scenario: No overlays is identical to loading the base alone

- **WHEN** the catalog is loaded with no overlays
- **THEN** the result is identical to loading the base source by itself
