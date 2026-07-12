## ADDED Requirements

### Requirement: Codex is a supported adapter

Codex SHALL be a supported tool adapter alongside Claude and OpenCode, selected
when a resource targets `codex`. The default target set SHALL remain Claude and
OpenCode so existing configs are unaffected, and Codex projection SHALL be
opt-in per resource. The Codex adapter SHALL be built on the shared adapter
contract rather than duplicating the Claude or OpenCode control flow.

#### Scenario: Codex target is recognized

- **GIVEN** a resource that lists `codex` in its targets
- **WHEN** the config loads and plan runs
- **THEN** the Codex adapter produces its changes and unknown-target validation still rejects other unknown tools

#### Scenario: Default targets exclude Codex

- **GIVEN** a resource with no explicit targets
- **WHEN** it is projected
- **THEN** it targets Claude and OpenCode only, leaving Codex opt-in
