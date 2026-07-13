# Design — onto close archive-move rollback

## Approach

In `internal/ontocli/close.go` `runClose`, the tail is:
```
st.Archived = true
Save(statePath, st)
MkdirAll(archive parent)
Rename(changeDir, archiveDir)
```
Wrap the destructive steps so a failure after marking `archived` restores
consistency:
```
st.Archived = true
if err := Save(statePath, st); err != nil { return err }
rollback := func() { st.Archived = false; _ = Save(statePath, st) }
if err := MkdirAll(archive parent); err != nil { rollback(); return err }
if err := Rename(changeDir, archiveDir); err != nil { rollback(); return err }
```
On the success path nothing changes: the rename moves the change dir (with its
`archived: true` state file) into the archive location.

## Why save-then-rollback rather than rename-first

The `archived` flag lives in `onto-state.yaml` inside the change dir. Saving it
first keeps the on-success record co-located so the atomic rename carries it
into the archive with no second write at the new path. The only cost is the
error-path rollback, which this change adds. (Deriving archived-ness from the
directory's location instead of a flag would remove the window entirely but is a
larger redesign — out of scope.)

## Test

Force `Rename` to fail deterministically (e.g. pre-create the dated archive
target as a file, or make the archive parent unwritable) and assert: `runClose`
returns an error AND the change's `onto-state.yaml` still has `archived: false`
(rolled back) and the change dir is still at its original path.

## Alternatives
- Location-derived archived state — rejected as too large for this slice.
