## ADDED Requirements

### Requirement: Claude marketplace declaration model

Plugin marketplaces SHALL be declarable as `[marketplaces.claude.<name>]` tables.
Marketplaces are Claude-only (OpenCode has no marketplace concept). Each table
SHALL carry a `source` type and its type-specific locator:

- `source = "github"` requires `repo` (`"owner/repo"`);
- `source = "url"` requires `url`;
- `source = "git-subdir"` requires `url` and `path`;
- `source = "directory"` requires `path`;
- `auto_update` (optional boolean).

The marketplace name SHALL be validated as a config key. An unrecognized `source`
type, or a missing required locator field for the declared type, SHALL be
rejected at load naming the marketplace. `settings.claude.extraKnownMarketplaces`
SHALL be rejected as reserved (homonto manages that structure).

#### Scenario: Parse a github marketplace

- **GIVEN** `[marketplaces.claude.official]` with `source = "github"` and `repo = "anthropics/claude-plugins"`
- **WHEN** the config is parsed
- **THEN** it yields a Claude marketplace `official` with a github source and that repo

#### Scenario: Unknown source type is rejected

- **GIVEN** `[marketplaces.claude.x]` with `source = "svn"`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the marketplace and the invalid source

#### Scenario: Missing locator for the source type is rejected

- **GIVEN** `[marketplaces.claude.x]` with `source = "github"` and no `repo`
- **WHEN** the config is parsed
- **THEN** it is rejected naming the marketplace and the missing field

#### Scenario: Reserved marketplace settings key rejected

- **GIVEN** a `settings.claude` key `extraKnownMarketplaces`
- **WHEN** the config is parsed
- **THEN** it is rejected as reserved
