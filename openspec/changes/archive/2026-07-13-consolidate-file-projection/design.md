# Design — consolidate file-projection

## High-level approach

Add `internal/adapter/fileproj` (the symlink analogue of `structproj`) and
migrate both adapters' three link namespaces onto it. Type-agnostic contract:
the adapter precomputes a flat `[]fileproj.Link` so the shared code never
branches on resource type.

### Contract API

```go
type Link struct { Dst, Src, Key, Inactive string }

func Project(tool string, links []Link, st *state.State, roots []string) ([]adapter.Change, error)
func Conflicts(links []Link, roots []string) error
func ApplyState(tool string, changes []adapter.Change, st *state.State, roots []string, fallbackDst func(key string) string) error
func ApplyLinks(tool string, links []Link, st *state.State, roots []string) error
func Observe(tool, prefix string, st *state.State) map[string]string
```

- **Project**: `link.Plan` over `{Dst:Src}`; per op — `Cur=="" &&
  Inactive!="" && IsManaged(Inactive)` → relocate (`update`, Old=Inactive,
  New="dst -> src"); `Cur==""` → `create` (New="dst -> src"); else → `update`
  (Old=op.Cur, **New=bare src**). Then adopt-unrecorded: readlink Dst, if it
  equals Src and state's Applied != Hash("dst -> src"), emit `adopt`. Emits **no
  deletes**. Does not sort (adapter's final sort handles it).
- **Conflicts**: `link.Plan` error-only precheck.
- **ApplyState**: for prefix-filtered changes — `adopt` → `st.Set(tool,key,
  New,Hash(New))`; `delete` → resolve dst (recordedDst else fallbackDst) →
  `link.Remove` + `st.Delete`.
- **ApplyLinks**: per link — if `Inactive!="" && IsManaged(Inactive)` →
  `link.Remove(Inactive)`; then `link.Link(Src,Dst)`; then `st.Set(tool,key,
  "dst -> src",Hash("dst -> src"))`.
- **Observe**: per recorded key of prefix — read `recordedDst(e.Desired)`,
  `os.Readlink`, on error continue, else `out[key]=Hash(dst+" -> "+target)`.

`recordedDst` and the `" -> "` separator move into `fileproj` (deleted from both
adapters' util.go).

### Adapter side (the only per-type code left, ~12 lines ×3)

Each adapter builds `skillFileLinks()/commandFileLinks()/subagentFileLinks()`
computing `Dst=filepath.Join(dir(scope), name[.md])`, `Src=source(entry)`,
`Key=prefix+name`, `Inactive= inactiveDir(scope)=="" ? "" :
filepath.Join(inactiveDir(scope), filepath.Base(Dst))`. The subagent builder
skips copy-mode entries. `fallbackDst` closures mirror the current user-scope
fallback.

Plan: three `Project` calls replace the six inline blocks. Apply file section,
**order preserved**: `ApplyState`×3 → `Conflicts`×3 → copy conflict precheck
(unchanged) → JSON writes (unchanged) → `ApplyLinks`×3 → `applyCopySubagents`
(unchanged). Observe: three `fileproj.Observe` calls; `subagentcopy.` stays
inline.

## Identity-preservation risks (from design investigation)

1. **Inactive `""` sentinel** must survive — never `filepath.Join("", name)`.
2. **`filepath.Base(dst)` unification** — `Join(inactive, Base(dst))` equals the
   current `name`(skills)/`name+".md"` forms; the skills (directory) case is the
   canary to confirm.
3. **relink asymmetry** — create/relocate emit New="dst -> src"; plain relink
   emits bare New=src with Old=op.Cur. Preserve exactly.
4. **hash string** — adopt/link/observe all hash `"dst -> src"` with the exact
   `" -> "` separator recordedDst cuts on. One shared constant.
5. **fail-fast conflict ordering** (biggest risk) — all `Conflicts` prechecks
   run BEFORE any JSON write or link creation; `ApplyState` (adopt/delete) runs
   before `Conflicts`, as today. `ApplyLinks` must never be the first place a
   conflict surfaces.
6. **inactive-prune keeps the `IsManaged` guard** so `link.Remove` only touches
   our own symlink.
7. **generic delete loop stays; fileproj plans no deletes** — no double-delete.
8. **Observe reads recorded dst, not current scope** — a pending scope switch
   leaves the applied link at the old location; continue on Readlink error.
9. **ApplyState per-prefix interleaving** is harmless (state is a keyed map).
10. **Old redaction** — relocate Old is a plain path (unredacted); only the
    generic-loop delete redacts. File keys never carry secrets.

## Migration order (each step green before the next)

0. Add `fileproj` + table-driven unit tests (isolated green).
1. claude skills (canary — directory case). Narrow inline adopt/delete loop to
   command./subagent. so skills aren't double-processed.
2. claude commands. 3. claude subagents (inline loop now empty → delete it;
   drop claude's dead recordedDst).
4-6. opencode skills → commands → subagents (drop opencode's dead recordedDst).
7. (separate change) copy-mode.

## Alternatives considered

- **Fold copy-mode in now** — rejected; different primitive (content-hash, local
  edits, prune-root guard) would muddy the Link API. Separate follow-up.
- **fileproj owns deletes (like structproj)** — rejected; file keys are pruned
  by the generic loop and don't share a document, so the generic loop stays the
  single delete source (no double-delete).
