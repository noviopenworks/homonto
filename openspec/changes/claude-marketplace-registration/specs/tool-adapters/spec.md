## ADDED Requirements

### Requirement: Claude marketplace projection

The Claude adapter SHALL project declared `[marketplaces.claude.<name>]` entries
to `extraKnownMarketplaces.<name>` in `settings.json`, via a managed key
namespace `marketplace.<name>`, surgically and idempotently. Specifically:

- desired: each declared marketplace contributes `marketplace.<name>` whose value
  is `{"source": {"source": <type>, <locator fields>}[, "autoUpdate": <bool>]}`;
- read-back: existing `extraKnownMarketplaces` members are read back as
  `marketplace.<name>` and excluded from the generic settings read-back;
- apply: the object is written at `extraKnownMarketplaces.<name>`, preserving
  unrelated `settings.json` keys and other marketplaces;
- prune: a de-declared marketplace deletes `extraKnownMarketplaces.<name>`;
- adoption: a pre-existing `extraKnownMarketplaces.<name>` equal to the desired
  value is adopted into state without rewriting the file;
- consecutive plans are byte-identical (deterministic).

#### Scenario: github marketplace projected

- **GIVEN** `[marketplaces.claude.official]` (`source = "github"`, `repo = "anthropics/claude-plugins"`)
- **WHEN** apply runs
- **THEN** `settings.json` `extraKnownMarketplaces.official.source` is `{"source":"github","repo":"anthropics/claude-plugins"}`, and unrelated keys are preserved

#### Scenario: De-declared marketplace is pruned

- **GIVEN** an `extraKnownMarketplaces.<name>` previously written and recorded by homonto, whose marketplace is no longer declared
- **WHEN** apply runs
- **THEN** `extraKnownMarketplaces.<name>` is deleted from `settings.json`

#### Scenario: Marketplace plan is deterministic

- **GIVEN** a declared marketplace with an `auto_update` flag
- **WHEN** `plan` runs twice consecutively
- **THEN** the two plans are byte-identical
