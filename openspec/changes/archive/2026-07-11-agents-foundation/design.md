## Context

Roadmap v2 foundation. Agents become first-class managed resources with
lifecycle metadata (version, mode) beyond v1's `[subagents.<name>]` symlinks.
This increment adds only the declaration model + a read-only `homonto agents
list`, deferring all mutation (add/update/pin/migrate), the lockfile,
compatibility checks, three-way-merge, and remote sources — mirroring how the
onto binary started read-only.

## Goals / Non-Goals

**Goals**: `[agents.<name>]` model (`Agent{Source,Version,Targets,Mode}`) +
validation reusing the existing `validSource`/`validateKey`/target checks; a
read-only `homonto agents list`.

**Non-Goals (this increment)**: any projection or file write for agents; the
lockfile/state; `add`/`update`/`pin`/`doctor`/`migrate`; compatibility checks;
three-way-merge/backup; remote sources; changing `[subagents.<name>]`.

## Decisions

### D1 — Model (`internal/config/config.go`)

```go
type Agent struct {
    Source  string   `toml:"source"`
    Version string   `toml:"version"`
    Targets []string `toml:"targets"`
    Mode    string   `toml:"mode"`
}
func (a Agent) TargetsOrAll() []string { if len(a.Targets)==0 { return []string{"claude","opencode"} }; return a.Targets }
func (a Agent) ModeOrDefault() string  { if a.Mode=="" { return "link" }; return a.Mode }
// Config gains: Agents map[string]Agent `toml:"agents"`
```

### D2 — Validation (`validateAgents`, called from Parse/Load)

For each `name, ag := range c.Agents`: `validateKey("agents", name)`;
`validSource(ag.Source)` (the existing builtin:/local: check — reject remote/
unknown); `ag.Mode ∈ {"", "copy", "link"}` else error naming agent+mode; each
target ∈ {claude, opencode}. Reuse the exact error-message style of
`validateResources`.

### D3 — `homonto agents list` (`internal/cli/agents.go`)

A parent `agentsCmd()` (Use `agents`, no RunE → shows help) with a `list`
subcommand. `list` reads `--config`, `config.Load(cfgPath)`, sorts agent names,
and prints one line per agent:
`<name>: <source>  version=<v|unpinned>  targets=<claude,opencode>  mode=<link|copy>`.
Empty → `No agents declared.`. Register `agentsCmd()` on the root next to the
other commands. Read-only: loads config only, never builds the engine or writes.

## Risks / Trade-offs

- **Model vs `[subagents]` overlap**: both describe agents, but `[subagents]` is
  the v1 symlink `Resource` (scope-based) and `[agents]` is the v2 lifecycle
  model (version/mode-based). They coexist; a later increment decides whether
  `[agents]` supersedes `[subagents]`. This increment keeps them independent and
  documents the distinction.
- **Read-only list of unrealized agents**: `list` shows declared intent, not
  installed state (no lockfile yet). The output labels are about declaration; a
  later `doctor`/`status` increment adds installed/version state.

## Migration Plan

Additive; `[agents]` optional. No migration.

## Open Questions

None for the foundation. Whether `[agents]` eventually subsumes `[subagents]` is
a later-increment decision.
