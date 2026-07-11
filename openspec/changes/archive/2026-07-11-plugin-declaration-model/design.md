## Context

Plugins are declared in `homonto.toml` as bare name lists and projected
minimally: Claude writes `enabledPlugins.<name> = true` (enable-only), OpenCode
appends `<name>` to the `plugin` array. Roadmap v1.2 expands this to
declarations with configuration. This change is v1.2's first increment: the
declaration-table model + enable/disable, deferring per-plugin `config` and
Claude marketplace registration to a follow-up.

The two tools' plugin systems genuinely differ (Claude: `enabledPlugins`
object keyed by `name@marketplace`, plus `extraKnownMarketplaces` and
`pluginConfigs`; OpenCode: a plain `plugin` string array with no per-plugin
config), so the model stays tool-scoped (`[plugins.claude.*]` vs
`[plugins.opencode.*]`) with tool-appropriate meaning for `source` — no unified
cross-tool abstraction (a stated roadmap non-goal).

## Goals / Non-Goals

**Goals**
- Declaration tables `[plugins.<tool>.<name>]` with required `source` +
  optional `enabled` (default true).
- Projectable disable (Claude `false`; OpenCode array removal), which the
  current model cannot express.
- Update both adapters + config validation + all existing plugin tests.

**Non-Goals (this increment)**
- Per-plugin `config` → Claude `pluginConfigs` (follow-up).
- Claude `extraKnownMarketplaces` registration (follow-up).
- OpenCode `config` handling (OpenCode has no native per-plugin config;
  follow-up decides warn/drop).
- Any migration shim for the old list form (pre-release breaking change).

## Decisions

### D1 — Model

`type Plugin struct { Source string \`toml:"source"\`; Enabled *bool
\`toml:"enabled"\` }`. `Plugins{ Claude map[string]Plugin \`toml:"claude"\`;
OpenCode map[string]Plugin \`toml:"opencode"\` }`. `Enabled` is a pointer so
"omitted" (nil → true) is distinguishable, though both nil and true mean
enabled. A helper `(Plugin).IsEnabled() bool` returns `Enabled == nil ||
*Enabled`.

### D2 — Validation

For each tool's plugins: `validateKey("plugins.<tool>", <declName>)` and reject
an empty `source`. Preserve the `settings.claude.enabledPlugins` and
`settings.opencode.plugin`/`mcp` reserved-key guards unchanged.

### D3 — Claude projection

In the desired-map builder, replace `out["plugin."+p] = \`true\`` with, for each
`name, pl := range c.Plugins.Claude`: `out["plugin."+name] =
mustJSON(pl.IsEnabled())` and project the value at `enabledPlugins[pl.Source]`.
The state/prune key stays `plugin.<name>`; the on-disk `enabledPlugins` key is
`pl.Source`. Disabled plugins now emit `false` (a real managed value) rather
than being absent — so `plan` shows disable, and apply writes it.

### D4 — OpenCode projection

For each `name, pl := range c.Plugins.OpenCode`: if `pl.IsEnabled()`, behave as
today but with the array value = `pl.Source` and state key `plugin.<name>`
(adopt/create as now). If `!pl.IsEnabled()`, ensure `pl.Source` is absent: if
present and managed in state, emit a prune (`delete`) change; else noop.

### D5 — Breaking change, no shim

The old `[plugins] claude = [...]` list no longer parses (the field is now a
map). Pre-release, acceptable. The change body documents the new form.

## Risks / Trade-offs

- **Decl-name vs source split** (state keyed by name, disk keyed by source):
  keeps state stable if a plugin is renamed at the source while the decl name is
  constant, and lets the OpenCode array hold the real package while state stays
  readable. Slight extra indirection in both adapters; covered by tests.
- **OpenCode disable = array removal** touches the shared array-membership prune
  path; must not remove entries homonto doesn't manage (guarded by state
  membership, like other prunes).
- **Test churn**: every existing plugin test moves from list to table form.
  Mechanical but broad; the plan calls it out explicitly.

## Migration Plan

None (pre-release). Any local `homonto.toml` using the list form must move to the
table form; documented in the change.

## Open Questions

None for this increment. `config` and marketplace handling are deferred, scoped
follow-ups.
