# config-model (delta)

## ADDED Requirements

### Requirement: local skill and command sources reject path traversal

A `[skills.<name>]` or `[commands.<name>]` `local:` source SHALL be a plain name
with no path components: `homonto` SHALL reject at load a `local:` source that is
empty, `.`, `..`, contains `/` or `\`, or is not equal to its own
`filepath.Base` — the same plain-name rule already applied to `[subagents.<name>]`
`local:` sources. A cleaned suffix that would escape the provider root SHALL be a
load error, never joined into a provider path.

#### Scenario: a traversal local source is rejected for skills and commands

- **GIVEN** a config with `[skills.x] source = "local:../../etc/x"` (or the same for a command)
- **WHEN** the config is loaded
- **THEN** it is rejected at load naming the skill/command and the invalid source, and nothing is projected

#### Scenario: a plain local source is accepted

- **GIVEN** `[skills.x] source = "local:x"`
- **WHEN** the config is loaded
- **THEN** it is accepted (resolves under the provider root)
