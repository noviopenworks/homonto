# framework-expansion

## ADDED Requirements

### Requirement: A local framework installs through the same validated path as a builtin

Config loading SHALL accept a framework whose source is `local:<path>`, where
`<path>` is a framework root (a `framework.toml` whose name equals the framework
key, plus its resources at framework-root-relative paths) resolved relative to
the config file. A `local:` framework MUST be validated through the same catalog
checks as a builtin (manifest schema, name-equals-key, resource-path existence,
dependency ranges) and its transitively-expanded resources MUST install through
the same projection and materialization path as a builtin framework's. Any other
non-builtin framework source MUST still fail loudly at load. A configuration
using only builtin frameworks MUST behave identically to before.

#### Scenario: A local framework's resource is installed by apply

- **GIVEN** a config declaring `[frameworks.myfw] source = "local:./myfw"` and a
  `./myfw` framework root providing a skill
- **WHEN** the change is applied
- **THEN** the skill is materialized and projected exactly as a builtin
  framework's skill would be

#### Scenario: A non-local, non-builtin framework source still fails

- **WHEN** a framework declares a source that is neither `builtin:` nor `local:`
- **THEN** loading fails loudly, unchanged from before

## MODIFIED Requirements

### Requirement: a non-builtin, non-local framework source fails at load

`homonto` SHALL reject at config load a `[frameworks.<name>]` declaration whose
source is neither a `builtin:<name>` source nor a `local:<path>` source, with a
clear error naming the framework and its source. Builtin and local frameworks
both expand and install; any other source (for example a bare name or a
`remote:` source) would expand nothing, so it SHALL be a load error rather than a
silent no-op.

#### Scenario: an unsupported framework source is rejected

- **GIVEN** a config with `[frameworks.onto] source = "onto"` (a bare name)
- **WHEN** the config is loaded
- **THEN** it is rejected naming the framework and the unsupported source, and nothing is installed

#### Scenario: a builtin framework source still loads

- **GIVEN** a config with `[frameworks.onto] source = "builtin:onto"`
- **WHEN** the config is loaded
- **THEN** it loads and the framework expands normally

#### Scenario: a local framework source loads

- **GIVEN** a config with `[frameworks.myfw] source = "local:./myfw"` and a matching framework root
- **WHEN** the config is loaded
- **THEN** it loads and the local framework expands like a builtin
