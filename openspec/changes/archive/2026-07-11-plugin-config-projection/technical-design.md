---
comet_change: plugin-config-projection
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-plugin-config-projection
status: final
---

# Plugin Config Projection — Technical Design

Deep refinement of `openspec/changes/plugin-config-projection/design.md`. v1.2
increment 2: project per-plugin `config` to Claude `pluginConfigs.<source>.options`
and reject OpenCode `config`. Real formats: `plugin-config-formats` memory.

## Model + validation (`internal/config/config.go`)

```go
type Plugin struct {
    Source  string         `toml:"source"`
    Enabled *bool          `toml:"enabled"`
    Config  map[string]any `toml:"config"` // NEW; non-sensitive per-plugin options
}
```

In the plugin validation loop, for OpenCode plugins reject a non-empty config:

```go
// (inside the per-tool loop, when tool.name == "plugins.opencode")
if len(pl.Config) > 0 {
    return nil, fmt.Errorf("parse config: %s plugin %q declares config, but OpenCode has no per-plugin config on disk (its plugins are a plain array); remove config", tool.name, declName)
}
```
Simplest: since the loop already knows `tool.name`, gate the check on
`tool.name == "plugins.opencode"`. Add `pluginConfigs` to the existing
`settings.claude` reserved-key rejection (next to `enabledPlugins`/`mcpServers`).

## Claude adapter (`internal/adapter/claude/claude.go`)

New managed namespace `pluginconfig.<source>` mirroring `plugin.`/`setting.`:

**desired()** (add after the existing plugin loop ~246):
```go
for _, pl := range c.Plugins.Claude {
    if len(pl.Config) > 0 {
        out["pluginconfig."+pl.Source] = mustJSON(map[string]any{"options": pl.Config})
    }
}
```

**current()** (read-back, ~476): add a pluginConfigs loop and exclude it from the
generic settings loop:
```go
for k, v := range objMembers(sj, "pluginConfigs") {
    out["pluginconfig."+k] = v
}
// in the generic settings loop's skip test:
if k == "mcpServers" || k == "enabledPlugins" || k == "pluginConfigs" { continue }
```

**apply write** (~686, add a case):
```go
case hasPrefix(c.Key, "pluginconfig."):
    sj, err = jsonutil.SetJSON(sj, "pluginConfigs."+jsonutil.EscapePath(trim(c.Key, "pluginconfig.")), val)
    sjChanged = true
```

**prune** (~608, add a case):
```go
case hasPrefix(c.Key, "pluginconfig."):
    sj, err = jsonutil.DeleteJSON(sj, "pluginConfigs."+jsonutil.EscapePath(trim(c.Key, "pluginconfig.")))
    sjChanged = true
```

**managed prefix** (`internal/adapter/claude/util.go` ~32): add `"pluginconfig."`
to the recognized-prefix slice so `managedPrefix`/orphan-prune recognize it.

Adoption needs no special case — the generic adopt path records the desired
value with `secret.Hash(Canonical(...))` of the on-disk value, and read-back
already surfaces `pluginConfigs.<k>` as `pluginconfig.<k>`.

### Why the value is the whole `{options: …}` object

Read-back yields each `pluginConfigs.<k>` member verbatim (an object
`{"options": {...}}`). Desired builds the identical shape. So the plan compares
like-for-like and is idempotent. Writing at `pluginConfigs.<source>` (not
`pluginConfigs.<source>.options`) with the `{options:…}` value keeps write and
read-back symmetric.

## Ordering / idempotency hazard (the one real risk)

`settings.json` now hosts THREE managed namespaces: top-level settings
(`setting.`), `enabledPlugins` (`plugin.`), and `pluginConfigs`
(`pluginconfig.`). The generic settings read-back loop iterates ALL top-level
members and must skip `mcpServers`, `enabledPlugins`, AND `pluginConfigs` — else
a `pluginConfigs` object is also surfaced as `setting.pluginConfigs`, desired
never has that key, and every plan proposes deleting it (non-idempotent) or
orphan-prunes it. The mixed test (enabled + config on the same plugin) locks this
in.

## Tests (`internal/config`, `internal/adapter/claude`)

- config: claude plugin with config parses (Config map populated); opencode
  plugin with config rejected (error names plugin); `settings.claude.pluginConfigs`
  rejected; existing enable/disable + dup-source + reserved-key tests still pass.
- claude: after apply, `pluginConfigs[source].options.<k>` on disk equals config;
  no-config plugin → no `pluginConfigs` entry; de-declared config pruned; adopt a
  pre-existing matching `pluginConfigs.<source>`; unrelated settings + other
  `pluginConfigs` entries preserved; consecutive plans byte-identical; a plugin
  with enabled+config projects BOTH `enabledPlugins[source]` and
  `pluginConfigs[source].options` with no `setting.pluginConfigs`/`setting.enabledPlugins`
  leaking into the plan.

## Verification

TDD RED→GREEN; full regression (build/test/-race/vet/gofmt/mod tidy). E2E via the
real `homonto` binary: a claude plugin with `config` → `apply` writes
`pluginConfigs.<source>.options`, second `plan` byte-identical; an opencode plugin
with `config` fails `plan` with the rejection message.

## Deferred

Claude marketplace registration (`extraKnownMarketplaces`) — needs a
marketplace-declaration model; the final v1.2 increment.
