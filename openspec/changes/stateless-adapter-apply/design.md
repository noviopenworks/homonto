# Design — stateless adapter Apply

## Approach

`Plan` opens by expanding entries:
```
skills, _ := c.ExpandedSkillEntriesForTool("claude"); a.skills = skills
commands, _ := ...; a.commands = commands
subagents, _ := ...; a.subagents = subagents
```
Extract this into `func (a *Adapter) expand(c *config.Config) error` and call it
at the top of BOTH `Plan` and `Apply`. Then `Apply` re-derives the same entries
from the config it is given, removing the "must call Plan first on this
instance" precondition.

### Interface

```go
Apply(cfg *config.Config, cs ChangeSet, res *secret.Resolver, st *state.State) error
```
- claude/opencode: `Apply` calls `a.expand(cfg)` first, then proceeds unchanged.
- codex: MCP-only (no file entries) — accepts `cfg` and ignores it (its Apply is
  already fully ChangeSet-driven via structproj).
- `engine.Apply`: `a.Apply(e.Cfg, cs, e.Resolver, e.State)`.

### Behavior identity

`Apply` receives the same `*config.Config` that `Plan` was given (the engine
holds one `e.Cfg`), so `expand(cfg)` reproduces the exact entries Plan set, and
every `*FileLinks()`/`copySubagentDesired`/fallback derivation is byte-identical.
The conformance suite + all adapter/engine tests pin this.

### Test-site migration

~61 direct `adapter.Apply(cs, res, st)` calls in adapter tests. Each is
preceded by `Plan(cfg)` so the config expression is already in scope; the change
is purely `Apply(cs, res, st)` → `Apply(<cfg>, cs, res, st)` with no assertion
change. Delegate the mechanical sweep, then verify the full suite.

## Risk

Low-per-site, broad. No behavior change; the safety net is the conformance suite
plus every adapter/engine test. Any diff is a migration slip, fixed in code.

## Alternatives

- Drive Apply purely from the ChangeSet (drop desired re-derivation) — rejected;
  ApplyLinks re-asserts every desired link (including already-correct ones), so
  it needs the full desired set, which the ChangeSet (changes only) lacks.
