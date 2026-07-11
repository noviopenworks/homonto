---
comet_change: plugin-declaration-model
role: technical-design
canonical_spec: openspec
---

# Plugin Declaration Model — Technical Design

Deep refinement of `openspec/changes/plugin-declaration-model/design.md`. First
increment of roadmap v1.2. The real tool plugin-config formats are recorded in
the `plugin-config-formats` project memory; this document nails the Go changes.

## Model (`internal/config/config.go`)

```go
// Plugin is one declared plugin. Source is the tool-native identifier:
// for claude the "name@marketplace" key used in enabledPlugins; for opencode
// the npm package / local plugin path placed in the `plugin` array.
type Plugin struct {
    Source  string `toml:"source"`
    Enabled *bool  `toml:"enabled"` // nil == true (default enabled)
}

// IsEnabled reports whether the plugin is enabled (default true when omitted).
func (p Plugin) IsEnabled() bool { return p.Enabled == nil || *p.Enabled }

type Plugins struct {
    Claude   map[string]Plugin `toml:"claude"`
    OpenCode map[string]Plugin `toml:"opencode"`
}
```

TOML:

```toml
[plugins.claude.claude-hud]
source  = "claude-hud@official"
enabled = true

[plugins.opencode.quota]
source = "@slkiser/opencode-quota"   # enabled defaults to true
```

## Validation (`config.go` Parse/Load, near the existing plugin loop ~line 409)

Replace the two `for _, p := range c.Plugins.Claude/OpenCode { validateKey... }`
loops with map-ranging equivalents:

```go
for _, tool := range []struct{ name string; m map[string]Plugin }{
    {"plugins.claude", c.Plugins.Claude},
    {"plugins.opencode", c.Plugins.OpenCode},
} {
    for declName, pl := range tool.m {
        if err := validateKey(tool.name, declName); err != nil {
            return nil, err
        }
        if strings.TrimSpace(pl.Source) == "" {
            return nil, fmt.Errorf("parse config: %s plugin %q has an empty source", tool.name, declName)
        }
    }
}
```

Keep the existing `settings.claude.enabledPlugins` / `settings.opencode.plugin` /
`mcp` reserved-key rejections unchanged.

## Claude projection (`internal/adapter/claude/claude.go` ~line 243)

Today:
```go
for _, p := range c.Plugins.Claude { out["plugin."+p] = `true` }
```
New:
```go
for name, pl := range c.Plugins.Claude {
    out["plugin."+name] = mustJSON(pl.IsEnabled()) // "true" or "false"
}
```
The desired-values map is keyed `plugin.<declName>`; the apply/prune path already
maps a `plugin.<key>` change onto `enabledPlugins.<key>` (claude.go ~474/615).
**Change needed**: the on-disk `enabledPlugins` key must be the plugin's
`source`, not the decl name. Two coherent options — pick the one that keeps
state/prune stable:

- **Option A (chosen)**: keep the state/desired key as `plugin.<source>` (use
  `pl.Source` where the current code used the bare name). Then existing
  read-back (`objMembers(sj,"enabledPlugins")` → `plugin.<k>`) and prune
  (`DeleteJSON enabledPlugins.<trim>`) already address the `source` key with no
  further change. The decl name is then only a TOML grouping label. This is the
  smallest, safest diff and keeps the enabledPlugins key == source on disk and
  in state.

Under Option A the Claude loop is:
```go
for _, pl := range c.Plugins.Claude {
    out["plugin."+pl.Source] = mustJSON(pl.IsEnabled())
}
```
Disabled plugins now emit `false` (a managed value), so `plan` shows the disable
and apply writes `enabledPlugins[source] = false`; prune still deletes the key
when the plugin is de-declared. Adoption of a pre-existing `enabledPlugins` key
is unchanged.

## OpenCode projection (`internal/adapter/opencode/opencode.go` ~line 263 and the apply path ~412)

Today ranges `[]string` and adopts/creates `plugin.<p>` with array value `p`.
New: range the map. For each `pl` with `pl.IsEnabled()`, behave exactly as today
using `pl.Source` as both the state key suffix and the array value
(`plugin.<source>`, value `pl.Source`) — preserving adopt/create/noop and the
"append without duplicating" guarantee. For a disabled plugin (`!IsEnabled()`):
if `arrayHas(doc,"plugin",pl.Source)` AND it is recorded in state (managed),
emit a prune/`delete` change removing it from the array; otherwise noop (never
touch an unmanaged entry). The apply path (~412) mirrors the same source-keyed
membership for writing the array.

Keying by `source` (not decl name) on the OpenCode side keeps the array value
and the state key identical — matching today's invariant (`name == array value`)
and minimizing prune-path risk. The decl name remains a TOML label.

> Note: this makes the effective key `source` for both adapters, so the decl
> name is purely organizational this increment. That is fine and is the least
> risky migration; a later increment that adds `config` can attach it by decl
> name without disturbing this.

## Test migration (`internal/{config,adapter/claude,adapter/opencode}/*_test.go`)

Every test that built `config.Plugins{Claude: []string{...}}` or
`OpenCode: []string{...}` moves to the map form
`map[string]config.Plugin{"name": {Source: "name"}}`. New assertions:

- config: parse table form; empty-source rejected; `enabled=false` → disabled;
  reserved keys still rejected.
- claude: `enabledPlugins[source]` == true (enabled) / false (disabled);
  unrelated keys preserved; consecutive plans byte-identical.
- opencode: enabled → source appended (no dup); disabled managed → removed;
  disabled-absent → noop; unmanaged entries preserved; adopt pre-existing.

## Verification

TDD RED→GREEN per task; then full regression (`go build ./...`,
`go test ./... -count=1`, `-race`, `go vet`, `gofmt -l .` empty, `go mod tidy`).
E2E: a `homonto.toml` with an enabled + a disabled claude plugin and an opencode
plugin → `homonto plan` shows the correct enable/disable; a second `plan` is
byte-identical (deterministic-plan requirement).

## Deferred (explicit follow-up increments)

- Per-plugin `config` → Claude `pluginConfigs.<source>.options`.
- Claude `extraKnownMarketplaces` registration from a marketplace declaration.
- OpenCode `config` handling (OpenCode has no native per-plugin config — decide
  warn vs. drop).
