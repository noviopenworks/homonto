## Why

onto #1/#2/#3a/#3b are archived: the `onto` binary can create a change (`onto
new`), advance it through gated phases (`onto advance`), and inspect it (`onto
status`). The workflow's terminal action — archiving a completed change — and its
dependency-resolution invariant are not yet enforced. Per the dual-binary design
the binary follows "archive and close rules" and ensures "dependencies are not
unresolved". This change (#3c, the final sub-increment of the onto workflow
engine) adds `onto close` and dependency-resolution gating.

## What Changes

- Add dependency-resolution helpers: `internal/ontostate.DepsResolved(root,
  deps) []string` returns the list of unresolved deps (a dep is resolved iff an
  archived change directory `docs/changes/archive/*-<dep>` exists under the
  workspace root). An empty or nil `deps` yields no unresolved (addresses the
  onto-skeleton OF-s1 nil/empty-`Deps` note: both mean "no dependencies").
- Add `onto close <change>`: archives a completed change. Preconditions
  (each failing case exits non-zero and archives NOTHING):
  - the framework-install gate passes and the change name is valid;
  - the change is at phase `close` (use `onto advance` to reach it);
  - every dep in the change's `onto-state.yaml` is resolved (else it names the
    unresolved deps);
  - the git worktree is clean (release-critical — a dirty or undeterminable
    worktree blocks the archive).
  On success it sets `archived: true` in the change's `onto-state.yaml`, then
  moves `docs/changes/<name>/` → `docs/changes/archive/<YYYY-MM-DD>-<name>/`
  (creating the archive dir), and reports the archived path. It REFUSES (no move)
  if the archive target already exists (no-clobber).
- `onto status` gains an "archived" indication for changes already under
  `docs/changes/archive/` is NOT in scope; status remains focused on active
  changes.
- This completes the onto workflow engine (create → advance → close). The `onto`
  binary now enforces the full transition + archive + dependency invariants.
  `onto doctor` (#4) and dual-binary packaging (#5) remain.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains `onto close <change>` (dependency-resolution-gated,
  clean-worktree-gated, no-clobber archive of a close-phase change) and the
  `DepsResolved` helper.

## Impact

- `internal/ontostate`: `DepsResolved(root string, deps []string) []string`
  (+ tests).
- `internal/ontocli`: new `closeCmd()` (`onto close`), registered on the root;
  reuses `gate`, `validChangeName`, `worktreeDirty`, and `ontostate.Save`.
- No new dependency. `onto` stays isolated from homonto's projection pipeline;
  shelling to `git` for the worktree check is the same allowance as `onto
  advance`.
- Completes the onto workflow engine (#3c of the onto binary work).
