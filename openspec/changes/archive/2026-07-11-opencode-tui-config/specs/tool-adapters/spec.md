## ADDED Requirements

### Requirement: OpenCode TUI file projection

The OpenCode adapter SHALL manage a second config file
`~/.config/opencode/tui.json` (in addition to `opencode.jsonc`), projecting each
`[tui.opencode]` key to a top-level key in `tui.json` via a `tui.<key>` state
namespace, surgically and idempotently. Specifically:

- `Plan` SHALL read `tui.json` and, for each declared `[tui.opencode]` key,
  produce a `tui.<key>` change (create / update / noop / adopt of a pre-existing
  matching key); a de-declared `tui.<key>` recorded in state SHALL be pruned;
- `ObserveHashes` SHALL read `tui.json` and hash the current value of each
  recorded `tui.<key>` still present;
- `Apply` SHALL write `tui.json` only when a `tui.<key>` change is applied,
  preserving unmanaged keys in the file, and SHALL delete a pruned key from it;
- unmanaged `tui.json` keys SHALL be preserved and consecutive plans SHALL be
  byte-identical.

Managing `tui.json` SHALL NOT alter how `opencode.jsonc` (MCPs, settings,
plugins) is managed.

#### Scenario: OpenCode TUI setting projected to tui.json

- **GIVEN** `[tui.opencode]` with `theme = "gruvbox"` against a `tui.json` with an unrelated key
- **WHEN** apply runs
- **THEN** `tui.json` `theme` is `"gruvbox"` and the unrelated key is preserved

#### Scenario: De-declared TUI setting is pruned from tui.json

- **GIVEN** a `tui.json` `theme` previously written and recorded by homonto, no longer declared
- **WHEN** apply runs
- **THEN** `theme` is deleted from `tui.json`

#### Scenario: TUI projection is idempotent and independent of opencode.jsonc

- **GIVEN** a config with both `[settings.opencode]` (→ opencode.jsonc) and `[tui.opencode]` (→ tui.json) keys, already applied
- **WHEN** `plan` runs twice consecutively
- **THEN** both plans are byte-identical and report no changes

#### Scenario: Adopt a pre-existing matching tui.json key

- **GIVEN** a `tui.json` already containing `theme = "gruvbox"` equal to the declared value, unrecorded in state
- **WHEN** apply runs
- **THEN** the key is adopted into state without rewriting `tui.json`
