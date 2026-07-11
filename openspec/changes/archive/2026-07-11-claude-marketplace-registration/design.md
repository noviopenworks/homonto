## Context

Final v1.2 increment. Claude loads a `name@marketplace` plugin only if the
marketplace is registered in `extraKnownMarketplaces` (see the
`plugin-config-formats` research). This adds a `[marketplaces.claude.<name>]`
declaration model and a `marketplace.<name>` managed namespace in the Claude
adapter — structurally identical to the `pluginconfig.` namespace shipped in v1.2
#2, so the read-back-exclusion idempotency discipline applies again.

## Goals / Non-Goals

**Goals**: `[marketplaces.claude.<name>]` model (source type + locator +
`auto_update`); validation; project to `extraKnownMarketplaces.<name>`
(desired/read-back/apply/prune/adopt/deterministic).

**Non-Goals**: OpenCode marketplaces (none exist); remote fetching; validating
the repo/url actually resolves; auto-installing plugins.

## Decisions

### D1 — Model (`internal/config/config.go`)

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

### D2 — Validation

Per claude marketplace: `validateKey("marketplaces.claude", name)`; `source` must
be one of the four types; the type's required locator present:
github→`Repo`, url→`URL`, git-subdir→`URL`+`Path`, directory→`Path`. Unknown
type or missing locator → error naming the marketplace. Add
`extraKnownMarketplaces` to the `settings.claude` reserved-key rejection.

### D3 — Claude adapter `marketplace.<name>` namespace (mirror `pluginconfig.`)

- **desired()**: for each `name, mk := range c.Marketplaces.Claude`,
  `out["marketplace."+name] = mustJSON(marketplaceValue(mk))` where
  `marketplaceValue` builds `{"source": {"source": mk.Source, <locator>}, ["autoUpdate": *mk.AutoUpdate]}`.
  The `source` sub-object includes only the locator fields relevant to the type
  (github→`repo`; url→`url`; git-subdir→`url`,`path`; directory→`path`) so the
  desired shape is canonical and stable. `autoUpdate` is emitted only when set.
- **current()**: `for k, v := range objMembers(sj, "extraKnownMarketplaces") { out["marketplace."+k] = v }`, and add `"extraKnownMarketplaces"` to the generic settings read-back skip set (now: mcpServers, enabledPlugins, pluginConfigs, extraKnownMarketplaces).
- **apply write**: `case hasPrefix(c.Key, "marketplace."): SetJSON(sj, "extraKnownMarketplaces."+EscapePath(trim(c.Key,"marketplace.")), val)`.
- **prune**: `case hasPrefix(c.Key, "marketplace."): DeleteJSON(sj, "extraKnownMarketplaces."+EscapePath(...))`.
- **managed prefix** (`util.go`): add `"marketplace."`.
- Adoption: generic path, no special case.

The value stored/compared is the whole `extraKnownMarketplaces.<name>` object, so
read-back and desired agree → idempotent (same discipline as pluginconfig).

## Risks / Trade-offs

- **Four managed namespaces in settings.json now** (settings, enabledPlugins,
  pluginConfigs, extraKnownMarketplaces): the generic settings read-back must
  skip all four; a missed skip re-surfaces the object as a phantom `setting.`
  key and churns the plan. A dedicated test (marketplace + a setting + a plugin
  in one file, re-plan byte-identical) locks it in.
- **Canonical `source` sub-object**: only type-relevant locator fields are
  emitted, so a github marketplace never carries an empty `url`/`path` that would
  differ from an adopted on-disk entry. `jsonutil.Canonical` handles key order.

## Migration Plan

Additive; `[marketplaces]` is optional. No migration.

## Open Questions

None. This completes v1.2.
