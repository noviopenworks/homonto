## ADDED Requirements

### Requirement: OpenCode TUI settings declaration

OpenCode TUI settings SHALL be declarable as a top-level `[tui.opencode]` table
(a map of key → value) whose entries homonto projects to
`~/.config/opencode/tui.json`. Each `[tui.opencode]` key SHALL be validated with
the same key-validation guard applied to other config keys. Claude TUI settings
SHALL NOT have a `[tui.claude]` table — they are ordinary top-level
`settings.json` keys already declarable under `[settings.claude]`.

#### Scenario: Parse OpenCode TUI settings

- **GIVEN** a config with `[tui.opencode]` containing `theme = "gruvbox"` and `scroll_speed = 3`
- **WHEN** the config is parsed
- **THEN** it yields an OpenCode TUI settings map with those two entries

#### Scenario: Invalid TUI key is rejected

- **GIVEN** a `[tui.opencode]` key that is an index-like or empty name (invalid config key)
- **WHEN** the config is parsed
- **THEN** it is rejected naming the key
