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
