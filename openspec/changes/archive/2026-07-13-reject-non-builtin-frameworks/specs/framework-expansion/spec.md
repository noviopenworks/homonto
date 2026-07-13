# framework-expansion (delta)

## ADDED Requirements

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
