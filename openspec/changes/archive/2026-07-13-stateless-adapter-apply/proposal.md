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
