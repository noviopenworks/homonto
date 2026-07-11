## Why

The plugin declaration model (v1.2 #1/#2) projects Claude plugin enable/disable
(`enabledPlugins`) and per-plugin config (`pluginConfigs`), but a Claude plugin
is identified by `name@marketplace` and Claude only loads it if that marketplace
is registered in `extraKnownMarketplaces`. homonto has no way to declare a
marketplace, so a declared plugin from a custom marketplace can't actually
resolve. This change (v1.2 #3, the final plugin-configuration increment) adds a
marketplace declaration model and projects it to Claude's `extraKnownMarketplaces`.
OpenCode has no marketplace concept (its plugins are npm packages / local files),
so marketplaces are Claude-only.

## What Changes

- Add a marketplace declaration model: `[marketplaces.claude.<name>]` tables with
  a `source` type and its type-specific locator:
  - `source = "github"` → `repo = "owner/repo"`;
  - `source = "url"` → `url = "https://…"`;
  - `source = "git-subdir"` → `url = "…"`, `path = "…"`;
  - `source = "directory"` → `path = "./…"`;
  - optional `auto_update` (bool).
- **Validation**: the marketplace name is a valid key; `source` is one of the
  four recognized types; the locator field required by that type is present
  (github→repo, url→url, git-subdir→url+path, directory→path); unknown source
  types and missing locators are rejected naming the marketplace.
- **Claude adapter** gains a managed key namespace `marketplace.<name>` projecting
  to `extraKnownMarketplaces.<name>` in `settings.json`:
  - desired: `marketplace.<name>` = `{"source": {"source": <type>, <locator>…}[, "autoUpdate": <bool>]}`;
  - read-back: `extraKnownMarketplaces` members → `marketplace.<name>` (and
    `extraKnownMarketplaces` excluded from the generic settings read-back);
  - apply writes the object at `extraKnownMarketplaces.<name>`;
  - prune deletes `extraKnownMarketplaces.<name>` when de-declared;
  - adoption of a pre-existing entry works like other keys;
  - `marketplace.` added to the managed-prefix set.
- `settings.claude.extraKnownMarketplaces` added to the reserved-settings guard.
- Surgical + idempotent; unrelated keys and other marketplaces preserved.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-model`: adds the `[marketplaces.claude.<name>]` declaration model
  (source + type-specific locator + optional auto_update; validated).
- `tool-adapters`: the Claude adapter projects marketplaces to
  `extraKnownMarketplaces.<name>` (surgical, idempotent, pruned, adoptable).

## Impact

- `internal/config/config.go`: `Marketplace` type, `Marketplaces` table on
  `Config`, validation.
- `internal/adapter/claude/claude.go`: `marketplace.` namespace across
  desired/current/apply/prune; `util.go` managed prefix.
- Tests in `internal/config` and `internal/adapter/claude`.
- No new dependency. No OpenCode change (marketplaces are Claude-only).
- **Completes roadmap v1.2 Plugin Configuration** (declare + enable/disable +
  config + marketplace).
