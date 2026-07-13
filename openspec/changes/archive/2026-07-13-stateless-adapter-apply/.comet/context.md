# Comet Design Handoff

- Change: stateless-adapter-apply
- Phase: design
- Mode: compact
- Context hash: 31fb7b98e857cf898b4d502f3c93baec1b86ca6ffee69fa65edb725387246c9f

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/stateless-adapter-apply/proposal.md

- Source: openspec/changes/stateless-adapter-apply/proposal.md
- Lines: 1-45
- SHA256: aedd44a28fe967370c044a85a5fafd256397879e93da3647484c3248b13924b5

```md
# Adapter.Apply derives from config, not mutable Plan-set instance state

## Why

Roadmap X2. `Adapter.Apply` reads the adapter's `skills`/`commands`/`subagents`
struct fields, which are populated ONLY by a prior `Plan` call on the same
instance (`a.skills = …` in `Plan`; read by the `*FileLinks()` builders,
`copySubagentDesired`, and the delete fallbacks in `Apply`). So `Apply` has a
hidden precondition — call `Plan` first on this exact instance — and silently
under-applies (links nothing) if that precondition is not met. The X2 problem
statement names this directly: "`Apply` reads mutable adapter fields set by a
prior `Plan`, not the plan alone."

## What Changes

Make `Apply` self-sufficient by giving it the config and deriving its file
entries the same way `Plan` does, so it no longer depends on instance state left
by a prior `Plan`:

- `Adapter.Apply` gains a leading `cfg *config.Config` parameter.
- Each adapter extracts a shared `expand(cfg) error` helper (the entry-expansion
  `Plan` already does) and calls it at the top of both `Plan` and `Apply`. The
  Codex adapter (MCP-only, no file entries) accepts `cfg` and ignores it.
- `engine.Apply` passes `e.Cfg` to each `a.Apply(...)`.

Behavior is identical: `Apply` receives the same config `Plan` was given, so the
re-derived entries — and thus the links it asserts — are exactly what they are
today; only the hidden precondition is removed.

## Impact

- **Specs:** `apply-pipeline` gains a requirement that applying a plan derives
  its managed file entries from the supplied config, not from state left by a
  prior planning call.
- **Behavior:** none — pure structural refactor, pinned by the conformance suite
  and every adapter/engine test.
- **Risk:** low-per-site but broad — an interface signature change touching 3
  adapters, the engine, and ~61 adapter test call sites (each already has the
  config in scope from its preceding `Plan`). Guarded by the full suite.

## Non-goals

- Driving `Apply` purely from the `ChangeSet` (dropping the desired-set
  re-derivation) — a larger rethink.
- Transaction journals (F42).

```

## openspec/changes/stateless-adapter-apply/design.md

- Source: openspec/changes/stateless-adapter-apply/design.md
- Lines: 1-49
- SHA256: 9d0b00655abb92704ebfeb7cdf794e3b6924f2262b79d0504db82bcdc527f410

```md
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

```

## openspec/changes/stateless-adapter-apply/tasks.md

- Source: openspec/changes/stateless-adapter-apply/tasks.md
- Lines: 1-14
- SHA256: 57a3a4df7c79965a8b1ff614f2d9a094a4f2cc082e0b28b6dab5fbda7b6af48d

```md
# Tasks — stateless-adapter-apply

## 1. Interface + adapters
- [ ] adapter.Adapter.Apply gains a leading cfg *config.Config param. Each of
      claude/opencode extracts an expand(cfg) helper called at the top of both
      Plan and Apply; codex accepts and ignores cfg. engine.Apply passes e.Cfg.

## 2. Test call sites
- [ ] Update the ~61 direct adapter Apply call sites to pass the config already
      in scope (from the preceding Plan). No assertion changes.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      conformance + all adapter/engine tests pass unchanged.

```

## openspec/changes/stateless-adapter-apply/specs/apply-pipeline/spec.md

- Source: openspec/changes/stateless-adapter-apply/specs/apply-pipeline/spec.md
- Lines: 1-21
- SHA256: 9537e0fb854782677361ee18bc3d914d541dfc592a85ea0c0d6a09179c11e7ba

```md
# apply-pipeline

## ADDED Requirements

### Requirement: Applying a plan derives managed file entries from config

An adapter's apply step SHALL derive its managed file-projection entries
(skills, commands, subagents) from the configuration supplied to it, not from
mutable instance state left by a prior planning call. Apply MUST be correct when
given the same configuration the plan was computed from, without depending on a
prior plan call having populated the adapter instance. The resulting on-disk
links, files, and recorded state MUST be identical to deriving them during
planning.

#### Scenario: Apply is correct without relying on prior-plan instance state

- **WHEN** an adapter applies a change set with the configuration it was planned
  from
- **THEN** it derives its managed file entries from that configuration and
  produces the same links, files, and state as before — with no hidden
  dependence on instance fields set by a prior plan call

```
