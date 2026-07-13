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
