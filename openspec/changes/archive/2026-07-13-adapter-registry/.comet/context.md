# Comet Design Handoff

- Change: adapter-registry
- Phase: design
- Mode: compact
- Context hash: 8a8c3d712abb455f8b03c27cf3f747d27378b2abf9ec520d50b9ab1dd79f86ca

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/adapter-registry/proposal.md

- Source: openspec/changes/adapter-registry/proposal.md
- Lines: 1-46
- SHA256: 64c7c44b44e58a6d2555fa1d0c726e5778368127b34b25adb33f70bd9b7c3acf

```md
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

```

## openspec/changes/adapter-registry/design.md

- Source: openspec/changes/adapter-registry/design.md
- Lines: 1-63
- SHA256: 1174851303cda5117f944824262be00c4ecc156ff3b492f624e09512c35d6abc

```md
# Design — adapter registry

## Package `internal/adapter/registry`

```go
type Deps struct {
    Home, ContentDir, ProjectRoot                          string
    CatalogDir, CommandCatalogDir, SubagentCatalogDir      string
    RemoteSubagentDir                                      string
}
type Factory func(Deps) adapter.Adapter

type Registry struct { order []string; factories map[string]Factory }
func New() *Registry
func (r *Registry) Register(id string, f Factory)   // panics on duplicate id
func (r *Registry) Build(d Deps) []adapter.Adapter  // in registration order

func Builtins() *Registry  // registers claude, opencode, codex (in that order)
```

`Builtins` is the one place built-ins are wired:
```go
r := New()
r.Register("claude", func(d Deps) adapter.Adapter {
    return claude.New(d.Home, d.ContentDir).
        WithProjectRoot(d.ProjectRoot).WithCatalogRoot(d.CatalogDir).
        WithCommandCatalogRoot(d.CommandCatalogDir).
        WithSubagentCatalogRoot(d.SubagentCatalogDir).
        WithRemoteSubagentRoot(d.RemoteSubagentDir)
})
r.Register("opencode", /* same options */)
r.Register("codex", func(d Deps) adapter.Adapter { return codex.New(d.Home) })
return r
```
No global mutable state / init() ordering: `Builtins()` returns a fresh
registry each call, so tests are hermetic. The `registry` package imports
`adapter`, `claude`, `opencode`, `codex`; those import `adapter` but not
`registry`, so there is no cycle.

## Engine wiring

`engine.Build` replaces the hardcoded `[]adapter.Adapter{...}` with:
```go
Adapters: registry.Builtins().Build(registry.Deps{
    Home: home, ContentDir: contentDir, ProjectRoot: projectRoot,
    CatalogDir: catalogDir, CommandCatalogDir: commandCatalogDir,
    SubagentCatalogDir: subagentCatalogDir, RemoteSubagentDir: remoteSubagentDir,
}),
```
Same three adapters, same order, same options → behavior-identical.

## Test

- registry unit test: `Builtins().Build(deps)` returns 3 adapters whose `Name()`
  is claude/opencode/codex in order; `Register` on a duplicate id panics.
- The engine + conformance + adapter suites pin behavior identity.

## Alternatives
- Global self-registration via `init()` + blank imports — rejected; global
  mutable state and import-for-side-effect are harder to test and reason about
  than an explicit `Builtins()`.
- A distinct `ToolID` value type — deferred; keys are the existing `Name()`
  strings to avoid churn. F34's type generalization is out of scope.

```

## openspec/changes/adapter-registry/tasks.md

- Source: openspec/changes/adapter-registry/tasks.md
- Lines: 1-13
- SHA256: 93a01d77edde7809a3ca542770f56597620c58bf347127cb966b3b4d623fe920

```md
# Tasks — adapter-registry

## 1. Registry package
- [ ] Add internal/adapter/registry: Deps, Factory, Registry (Register/Build in
      order, dup-panic), Builtins() registering claude/opencode/codex. Unit tests
      (Build yields the 3 in order; Register dup panics).

## 2. Engine wiring
- [ ] engine.Build constructs Deps and calls registry.Builtins().Build(deps);
      remove the hardcoded adapter literal. Engine + conformance suites green.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/adapter-registry/specs/adapter-contract/spec.md

- Source: openspec/changes/adapter-registry/specs/adapter-contract/spec.md
- Lines: 1-25
- SHA256: cda6702b51bb9dc146130540d7a0a890c4c4ecd6a08813780dac2fbe6f573059

```md
# adapter-contract

## ADDED Requirements

### Requirement: The engine sources adapters from a tool-id-keyed registry

The engine SHALL construct its set of tool adapters from a tool-id-keyed
registry of adapter factories, rather than a hardcoded list bound to each
adapter's concrete constructor. Registering a factory under a tool id MUST be
the only step required to add a built-in adapter; the engine MUST build every
registered adapter, in a deterministic order, passing each the shared
construction dependencies. Building from the registry MUST yield the same
adapters, with the same options, as the prior hardcoded wiring.

#### Scenario: Engine builds every registered adapter

- **WHEN** the engine builds its adapters
- **THEN** it builds one adapter per registered tool id, in registration order,
  each constructed from the shared dependencies — identical to the prior
  hardcoded set

#### Scenario: Adding an adapter is a registration

- **WHEN** a new adapter factory is registered under a new tool id
- **THEN** the engine includes it with no change to the engine's build logic

```
