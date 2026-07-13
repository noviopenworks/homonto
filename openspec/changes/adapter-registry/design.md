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
