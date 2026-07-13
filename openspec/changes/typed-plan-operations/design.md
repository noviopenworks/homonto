# Design — typed, validated plan operations

## Approach

### Typed Action
In `internal/adapter/adapter.go`:
```go
type Action string
const (
    ActionCreate Action = "create"
    ActionUpdate Action = "update"
    ActionDelete Action = "delete"
    ActionNoop   Action = "noop"
    ActionAdopt  Action = "adopt"
)
func (a Action) Valid() bool { … one of the five … }
```
`Change.Action` changes from `string` to `Action`. Because the constants keep
the same underlying string values, existing `Change{Action: "create"}`
construction and `c.Action == "noop"` comparison keep compiling (untyped string
constants convert to `Action`). No adapter construction site must change; the
constants are available for new code.

### Validation
```go
func (cs ChangeSet) Validate(knownTools map[string]bool) error
```
- error if `!knownTools[cs.Tool]` — "unknown tool" (fail-closed; today silently
  skipped);
- error if any `!c.Action.Valid()` — "unknown action %q for key %q".

### Engine wiring
`engine.Apply` builds `knownTools` from its registered adapters and calls
`cs.Validate(knownTools)` for every set at the very top — before the secret
resolve loop, `materializeRemotes`, `materializeCatalog`, and any adapter Apply.
An error aborts with no side effect. The existing `byName[cs.Tool]` skip stays
as defensive code but can no longer hide an unknown tool (validation already
errored).

## Identity / safety
- No legal plan changes behavior: every action a real adapter emits is one of
  the five; every set's tool is a registered adapter, so `Validate` passes.
- The new failure is strictly a previously-silent drop becoming a clear abort.

## Migration
0. Typed Action + Valid() + ChangeSet.Validate + unit tests (adapter package).
1. engine.Apply validates first + engine test (unknown tool aborts; unknown
   action aborts; legal plan applies). Full suite green.

## Alternatives
- Validate inside each adapter's Apply — rejected; the engine is the one choke
  point that sees every set and the registered-tool set, and it must abort
  before materialization, not per-adapter mid-apply.
