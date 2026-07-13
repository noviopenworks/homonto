# ToolID-keyed adapter registry — a new adapter is a registration, not an engine edit

## Why

Roadmap X3 (F33/F34). The set of built-in adapters is a hardcoded slice literal
inside `engine.Build`:
```go
Adapters: []adapter.Adapter{ claude.New(...).With...(), opencode.New(...).With...(), codex.New(home) }
```
Adding or removing an adapter means editing the engine's composition root, and
the engine is coupled to every adapter's concrete constructor and option
methods. The X3 exit gate calls for "a `ToolID`-keyed capability registry so a
new adapter is a registration, not a repo-wide edit."

## What Changes

Introduce `internal/adapter/registry`:

- `Deps` — the bundle of construction inputs an adapter may need (home, content
  dir, project root, the three catalog roots, the remote-subagent root).
- `Factory func(Deps) adapter.Adapter` and a `Registry` that registers factories
  by tool id and builds the adapter list in registration order.
- `Builtins() *Registry` — the single place the built-in adapters are registered
  (claude, opencode, codex). Adding an adapter is one registration line here, not
  an engine-logic edit.
- `engine.Build` constructs its `Deps` and calls `registry.Builtins().Build(deps)`
  instead of the hardcoded literal — identical adapters, identical order.

## Impact

- **Specs:** `adapter-contract` gains a requirement that the engine sources its
  adapters from a tool-id-keyed registry, so a new adapter is added by
  registration.
- **Behavior:** none — the same three adapters in the same order with the same
  options; a pure wiring refactor pinned by the engine + conformance + adapter
  suites.
- **Risk:** low — a new package plus a one-site engine wiring change; the full
  suite is the safety net.

## Non-goals

- Decoupling the `Adapter` interface from the concrete `config.Config`/
  `secret.Resolver`/`state.State` types (the deeper F34 generalization).
- Global init()-based self-registration (kept explicit and testable instead).
- Any adapter behavior or a distinct `ToolID` value type (the registry keys on
  the existing tool-name strings that `Name()` returns).
