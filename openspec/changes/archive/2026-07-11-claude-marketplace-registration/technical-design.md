---
comet_change: claude-marketplace-registration
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-claude-marketplace-registration
status: final
---

# Claude Marketplace Registration — Technical Design

Final v1.2 increment. Adds a `[marketplaces.claude.<name>]` model and a
`marketplace.<name>` Claude-adapter namespace projecting to
`extraKnownMarketplaces.<name>`. Structurally identical to the `pluginconfig.`
namespace (v1.2 #2), so the read-back-exclusion idempotency discipline recurs.
Real format: `plugin-config-formats` memory.

## Model + validation (`internal/config/config.go`)

```go
type Marketplace struct {
    Source     string `toml:"source"`      // github | url | git-subdir | directory
    Repo       string `toml:"repo"`        // github
    URL        string `toml:"url"`         // url, git-subdir
    Path       string `toml:"path"`        // git-subdir, directory
    AutoUpdate *bool  `toml:"auto_update"` // optional
}
type Marketplaces struct { Claude map[string]Marketplace `toml:"claude"` }
// Config gains: Marketplaces Marketplaces `toml:"marketplaces"`
```

Validation (add a loop near the plugin validation): for each
`name, mk := range c.Marketplaces.Claude`:
- `validateKey("marketplaces.claude", name)`;
- switch `mk.Source`:
  - `"github"`: require `mk.Repo != ""`;
  - `"url"`: require `mk.URL != ""`;
  - `"git-subdir"`: require `mk.URL != ""` and `mk.Path != ""`;
  - `"directory"`: require `mk.Path != ""`;
  - default: error `unknown source %q` naming the marketplace.
  Missing locator → error naming the marketplace and the missing field.
Add `extraKnownMarketplaces` to the `settings.claude` reserved-key rejection.

## Claude adapter `marketplace.<name>` namespace (`internal/adapter/claude/claude.go`)

Helper building the canonical value:
```go
func marketplaceValue(mk config.Marketplace) map[string]any {
    src := map[string]any{"source": mk.Source}
    switch mk.Source {
    case "github":     src["repo"] = mk.Repo
    case "url":        src["url"] = mk.URL
    case "git-subdir": src["url"] = mk.URL; src["path"] = mk.Path
    case "directory":  src["path"] = mk.Path
    }
    out := map[string]any{"source": src}
    if mk.AutoUpdate != nil { out["autoUpdate"] = *mk.AutoUpdate }
    return out
}
```
Only type-relevant locator fields are emitted, so a github marketplace never
carries an empty `url`/`path` that would differ from an adopted disk entry.

- **desired()**: `for name, mk := range c.Marketplaces.Claude { out["marketplace."+name] = mustJSON(marketplaceValue(mk)) }`.
- **current()**: `for k, v := range objMembers(sj, "extraKnownMarketplaces") { out["marketplace."+k] = v }`, and extend the generic settings skip test to `k == "mcpServers" || k == "enabledPlugins" || k == "pluginConfigs" || k == "extraKnownMarketplaces"`.
- **apply write**: `case hasPrefix(c.Key, "marketplace."): sj, err = jsonutil.SetJSON(sj, "extraKnownMarketplaces."+jsonutil.EscapePath(trim(c.Key,"marketplace.")), val); sjChanged = true`.
- **prune**: `case hasPrefix(c.Key, "marketplace."): sj, err = jsonutil.DeleteJSON(sj, "extraKnownMarketplaces."+jsonutil.EscapePath(trim(c.Key,"marketplace."))); sjChanged = true`.
- **managed prefix** (`util.go`): add `"marketplace."`.
- Adoption: generic path, no special case.

## Idempotency hazard (same as pluginconfig)

`settings.json` now has FOUR managed namespaces. The generic settings read-back
loop must skip all four (`mcpServers`, `enabledPlugins`, `pluginConfigs`,
`extraKnownMarketplaces`) — else the marketplace object re-surfaces as a phantom
`setting.extraKnownMarketplaces`, desired never has it, and every plan churns.
The mixed-namespace test (a setting + a plugin + a pluginConfig + a marketplace
in one settings.json, re-plan byte-identical) locks it in.

## Tests

- config: parse a github marketplace (fields populated); unknown source rejected;
  missing repo (github) rejected; missing url/path (git-subdir) rejected;
  `settings.claude.extraKnownMarketplaces` rejected; existing plugin/config tests
  still pass.
- claude: after apply, `extraKnownMarketplaces[name].source == {"source":"github","repo":…}`;
  a `url`/`directory`/`git-subdir` marketplace projects the right locator;
  `autoUpdate` present only when set; de-declared marketplace pruned; adopt a
  pre-existing matching entry; unrelated settings + other marketplaces preserved;
  consecutive plans byte-identical; a settings.json with a setting + plugin +
  pluginConfig + marketplace re-plans byte-identical (no namespace leaks).

## Verification

TDD RED→GREEN; full regression. E2E via the real `homonto` binary: a github
`[marketplaces.claude.official]` + `[plugins.claude.hud] source="hud@official"` →
`apply` writes `extraKnownMarketplaces.official.source` and
`enabledPlugins["hud@official"]`; second `plan` byte-identical.

## Completes v1.2

declare (#1) + enable/disable (#1) + config (#2) + marketplace (#3).
