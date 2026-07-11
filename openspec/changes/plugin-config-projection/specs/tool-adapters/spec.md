## ADDED Requirements

### Requirement: Claude plugin config projection

The Claude adapter SHALL project a declared Claude plugin's `config` to
`pluginConfigs.<source>.options` in `settings.json`, via a managed key namespace
`pluginconfig.<source>`, surgically and idempotently. Specifically:

- desired state: each Claude plugin with a non-empty `config` contributes
  `pluginconfig.<source>` whose value is `{"options": <config>}`;
- read-back: existing `pluginConfigs` members are read back as
  `pluginconfig.<key>` and are excluded from the generic settings read-back;
- apply: the `{options: …}` object is written at `pluginConfigs.<source>`,
  preserving unrelated `settings.json` keys and other `pluginConfigs` entries;
- prune: a de-declared plugin config deletes `pluginConfigs.<source>`;
- adoption: a pre-existing `pluginConfigs.<source>` equal to the desired value is
  adopted into state without rewriting the file;
- consecutive plans are byte-identical (deterministic).

A Claude plugin without a `config` (or an empty one) contributes no
`pluginConfigs` entry. OpenCode has no per-plugin config projection (a `config`
on an OpenCode plugin is rejected at load).

#### Scenario: Claude plugin config projected under options

- **GIVEN** a Claude plugin `[plugins.claude.hud]` with `source = "hud@official"` and `config = { api_endpoint = "https://x" }`
- **WHEN** apply runs
- **THEN** `settings.json` `pluginConfigs["hud@official"].options.api_endpoint` is `"https://x"`, and unrelated keys are preserved

#### Scenario: Plugin without config projects no pluginConfigs entry

- **GIVEN** a Claude plugin with no `config`
- **WHEN** apply runs
- **THEN** no `pluginConfigs` entry is written for it

#### Scenario: De-declared plugin config is pruned

- **GIVEN** a `pluginConfigs.<source>` previously written and recorded by homonto, whose plugin no longer declares `config`
- **WHEN** apply runs
- **THEN** `pluginConfigs.<source>` is deleted from `settings.json`

#### Scenario: Plugin config plan is deterministic

- **GIVEN** a Claude plugin with a multi-key `config`
- **WHEN** `plan` runs twice consecutively
- **THEN** the two plans are byte-identical
