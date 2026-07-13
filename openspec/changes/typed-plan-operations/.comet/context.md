# Comet Design Handoff

- Change: typed-plan-operations
- Phase: design
- Mode: compact
- Context hash: a0b26a83aacf16b884322b4ea353b3bc5572030e937ca4f34597e172c72f4773

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/typed-plan-operations/proposal.md

- Source: openspec/changes/typed-plan-operations/proposal.md
- Lines: 1-53
- SHA256: a48fd8aa585a8f2b0bdd03465193fde9f6cf02ef767d3297712ca8754ddec983

```md
# Typed, validated plan operations (fail-closed on unknown action/tool)

## Why

Roadmap X2 (F41), typed-operations slice. `adapter.Change.Action` is a bare
`string` whose five legal values live only in a doc comment; adapters construct
them as string literals at dozens of sites, and the engine consumes them with
`== "adopt"` comparisons. Nothing validates a change set before apply, so:

- an **unknown tool** change set is **silently skipped** (`engine.Apply`
  `byName[cs.Tool]` → `if !ok { continue }`) — a typo'd or stale tool name
  vanishes with no error;
- an **unknown action** falls through every switch: the secret-resolve loop
  treats it as a value change, and each adapter's Apply switch ignores it — so a
  bug (or a future/rolled-back binary reading an unexpected plan) silently
  no-ops instead of failing closed.

For a tool that mutates a user's config files, a malformed operation must abort
the apply, not be quietly dropped.

## What Changes

- Make `adapter.Action` a defined type with exported constants
  (`ActionCreate`/`ActionUpdate`/`ActionDelete`/`ActionNoop`/`ActionAdopt`) and
  an `Action.Valid()` method. `Change.Action` becomes `Action`. Existing string
  literals remain assignable, so this is low-churn.
- Add `ChangeSet.Validate(knownTools) error` — rejects an unknown action and a
  tool not backed by a registered adapter.
- `engine.Apply` calls `Validate` for every set **first**, before any secret
  resolution, remote/catalog materialization, or adapter write — fail-closed
  with a clear error naming the offending tool/action. Unknown-tool sets are now
  an error, not a silent skip.

## Impact

- **Specs:** `apply-pipeline` gains a requirement that apply validates every
  operation's action and tool before any side effect, aborting on an unknown
  one.
- **Behavior:** the only observable change is that a previously-silent
  unknown-tool/action set now aborts apply with an error. All legal plans are
  unaffected (every action a real adapter emits is valid; every tool a set
  carries is a registered adapter).
- **Risk:** low — additive validation on a fail-closed path, plus a
  non-breaking type refinement. Covered by new engine + adapter unit tests and
  the full existing suite.

## Non-goals

- Making `Apply` stateless (not reading adapter fields set by a prior `Plan`) —
  the deeper X2 immutability work.
- Transaction journals, versioned staging trees, close/archive validation
  (F42/F47/F4/F18) — later X2 slices.
- Typing `Key`/`Old`/`New` payloads.

```

## openspec/changes/typed-plan-operations/design.md

- Source: openspec/changes/typed-plan-operations/design.md
- Lines: 1-53
- SHA256: 2743c00d4d360c73905600efd86f1d7c741742a3b0915aeeebc8c61547c36d42

```md
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

```

## openspec/changes/typed-plan-operations/tasks.md

- Source: openspec/changes/typed-plan-operations/tasks.md
- Lines: 1-13
- SHA256: 2eaa28e47afc0b79855ee817aeda004f437e4b358810bb09338f6c1479fbace3

```md
# Tasks — typed-plan-operations

## 1. Typed action + validation
- [ ] adapter.Action defined type + constants + Valid(); Change.Action is Action.
      ChangeSet.Validate(knownTools) rejects unknown action/tool. Unit tests.

## 2. Engine fail-closed wiring
- [ ] engine.Apply validates every set first (before resolve/materialize/write),
      aborting on unknown tool or action. Engine tests: unknown tool aborts,
      unknown action aborts, legal plan applies unchanged.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/typed-plan-operations/specs/apply-pipeline/spec.md

- Source: openspec/changes/typed-plan-operations/specs/apply-pipeline/spec.md
- Lines: 1-31
- SHA256: 1e0efd1e8c641e3169a8e8cb41deed604586d7da0218d5693c1a76abbb8f0011

```md
# apply-pipeline

## ADDED Requirements

### Requirement: Apply validates every operation before any side effect

`Apply` SHALL validate every planned change set before performing any secret
resolution, remote or catalog materialization, or adapter write. A change set
whose tool is not a registered adapter, or that contains an operation whose
action is not one of the defined operations (create, update, delete, noop,
adopt), MUST abort the apply with an error naming the offending tool or action,
leaving no file or state mutated. Legal plans — every operation a registered
adapter emits — MUST be unaffected.

#### Scenario: Unknown tool aborts apply

- **WHEN** a change set names a tool that is not a registered adapter
- **THEN** apply aborts with an error and performs no write (the set is not
  silently skipped)

#### Scenario: Unknown action aborts apply

- **WHEN** a change set contains an operation whose action is not one of the
  defined operations
- **THEN** apply aborts with an error naming the offending action and performs
  no write

#### Scenario: Legal plan applies unchanged

- **WHEN** every change set carries a registered tool and only defined actions
- **THEN** validation passes and apply proceeds exactly as before

```
