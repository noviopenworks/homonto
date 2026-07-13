# onto close: leave nothing archived when the archive move fails

## Why

Roadmap X2 (F4). The `onto close` spec says every failing case "archives
NOTHING", but the implementation sets `archived: true` and saves
`onto-state.yaml` **before** the `os.Rename` that moves the change into
`docs/changes/archive/`. If the rename fails (e.g. a permission error on the
archive parent, or a cross-device boundary), the change is left with
`archived: true` in state but still sitting at its original path — a
marked-archived-but-not-moved inconsistency that contradicts the spec's
"archives nothing on failure" guarantee and confuses later `onto` commands.

## What Changes

Make the two-step mutation consistent on the error path: if the archive move
fails, roll the `archived` flag back to `false` (and re-save state) so a failed
close leaves the change fully un-archived — exactly the "archives NOTHING"
contract. On success the behavior is unchanged (`archived: true`, moved,
reported). The state file lives inside the change dir, so a successful rename
carries the `archived: true` record into the archive location as before; the
rollback path re-saves the still-present in-place state file.

## Impact

- **Specs:** the `onto close archives a completed change` requirement is
  clarified to state that a failed archive move leaves `archived` unset.
- **Behavior:** only the failure path changes — a rename failure now leaves
  `archived: false` instead of a stale `true`. Success path unchanged.
- **Risk:** low — a localized error-path rollback in `runClose`, covered by a
  new failure-injection test (rename made to fail) plus the existing onto close
  suite.

## Non-goals

- Full crash-safety of the flag+move (a process kill between the save and the
  rename still leaves a window — closing that needs location-derived archived
  state, a larger change). This fixes the deterministic error path.
- Broader X2 (stateless Apply, transaction journals).
