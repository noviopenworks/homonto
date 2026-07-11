## Why

Roadmap v1.3 (Tool TUI Configuration) manages Claude/OpenCode TUI-related config.
Research (see the `tui-config-formats` notes) shows Claude's TUI settings —
`theme`, `statusLine`, `tui`, `editorMode`, etc. — are all top-level
`settings.json` keys already covered by homonto's existing `[settings.claude]`
projection, so Claude needs no new code. OpenCode is different: its TUI settings
live in a **separate file** `~/.config/opencode/tui.json` (theme, keybinds,
scroll_speed, diff_style, mouse, …) that homonto does not currently write — it
only writes `opencode.jsonc`. This change adds an OpenCode `tui.json` projection
target: the first case of homonto managing **two** config files for one adapter.

## What Changes

- Add a top-level `[tui.opencode]` config table (a `map[string]any`) whose keys
  are written to `~/.config/opencode/tui.json`. A top-level `[tui.<tool>]` table
  (rather than nesting under `[settings.opencode]`, which specifically means
  `opencode.jsonc`) cleanly signals "the separate TUI file." Claude TUI settings
  intentionally have **no** `[tui.claude]` — they are top-level `settings.json`
  keys already handled by `[settings.claude]` (documented; avoids two ways to set
  one key). This diverges from the roadmap's illustrative `[settings.opencode.tui]`
  because TUI is a distinct file, so it earns a distinct table.
- **Validation**: each `[tui.opencode]` key is validated with the same key guard
  as other config keys.
- **OpenCode adapter** gains a second managed file `tui.json` and a `tui.<key>`
  state namespace, mirroring how `opencode.jsonc` settings are handled:
  - `Plan` reads `tui.json` and projects each `[tui.opencode]` key as
    `tui.<key>` (create/update/noop/adopt), and orphan-prunes de-declared
    `tui.<key>`;
  - `ObserveHashes` reads `tui.json` and hashes recorded `tui.<key>` values;
  - `Apply` writes `tui.json` (surgically; unmanaged keys preserved) only when a
    `tui.<key>` actually changed, and deletes a pruned key from it.
- Surgical + idempotent; unrelated `tui.json` keys preserved; consecutive plans
  byte-identical.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-model`: adds the `[tui.opencode]` declaration (map of TUI settings).
- `tool-adapters`: the OpenCode adapter manages a second config file,
  `~/.config/opencode/tui.json`, projecting `[tui.opencode]` keys (surgical,
  idempotent, pruned, adoptable).

## Impact

- `internal/config/config.go`: `TUI` type (`OpenCode map[string]any`), `Config.TUI`,
  validation.
- `internal/adapter/opencode/opencode.go`: a `tuiFile()` path + `tui.<key>`
  namespace across `Plan`, `ObserveHashes`, `Apply` (second-doc read/write) and
  the orphan-prune declared set.
- Tests in `internal/config` and `internal/adapter/opencode`.
- No new dependency. No Claude adapter change (Claude TUI already covered by
  `[settings.claude]`).
- Establishes the "two managed files per adapter" pattern for future TUI/keybind
  increments.
