## Context

onto #1/#2/#3a/#3b are archived: the binary can create (`onto new`), advance
(`onto advance`), and inspect (`onto status`) a change. #3c adds the terminal
`onto close` (archive) and the dependency-resolution invariant, completing the
onto workflow engine.

## Goals / Non-Goals

**Goals:** `ontostate.DepsResolved`; `onto close <change>` archiving a close-phase
change gated on resolved deps + a clean worktree, no-clobber, setting
`archived: true` and moving the dir into `docs/changes/archive/<date>-<name>/`.

**Non-Goals:** `onto doctor` (#4); dual-binary packaging (#5); spec-sync on
archive (onto's docs/ specs are the change's own; no main-spec merge like comet);
status listing archived changes; homonto/isolation changes.

## Decisions

**D1 — `DepsResolved(root, deps) []string` in `ontostate`.** For each dep,
resolved iff `filepath.Glob(filepath.Join(root,"docs","changes","archive",
"*-"+dep))` returns ≥1 match (the dep was archived under a date-prefixed dir).
Return the unresolved subset (nil/empty deps → empty result, in order). This
subsumes the onto-skeleton OF-s1 nil/empty-`Deps` note: both mean "no deps".

**D2 — `closeCmd()`: gate → validate → load → phase → deps → dirty → no-clobber
→ archive.** `onto close <change>` (ExactArgs(1) + `--dir` default "."):
1. `gate(root)`; on error return it, archive nothing.
2. `validChangeName(name)`; on error return it.
3. `changeDir := <root>/docs/changes/<name>`; `st, err := ontostate.Load(<dir>/
   onto-state.yaml)` (error if missing/invalid).
4. If `st.Phase != "close"` → non-zero "change is at %q; run `onto advance` until
   it reaches close" — nothing archived.
5. `unresolved := ontostate.DepsResolved(root, st.Deps)`; if non-empty → non-zero
   naming them — nothing archived.
6. `dirty, ok := worktreeDirty(root)` (reuse from advance.go); if `dirty || !ok`
   → non-zero "dirty/undeterminable worktree blocks close" — nothing archived.
7. `archiveDir := <root>/docs/changes/archive/<time.Now().Format("2006-01-02")>-
   <name>`; if it exists (os.Stat) → non-zero "archive target already exists" —
   nothing archived (no-clobber).
8. `st.Archived = true; ontostate.Save(<changeDir>/onto-state.yaml, st)` (so the
   archived state file records archived); `os.MkdirAll(<root>/docs/changes/
   archive)`; `os.Rename(changeDir, archiveDir)`. Report `"<change>: archived to
   <archiveDir>"`, exit 0.

Set `archived` BEFORE the move so the moved `onto-state.yaml` already carries it.
The move is `os.Rename` (atomic within the same filesystem).

**D3 — Reuse, no duplication.** `closeCmd` reuses `gate`, `validChangeName`,
`worktreeDirty`, and `ontostate.Save`/`Load` — no new git/gate logic.

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/ontostate` | `DepsResolved` | os, path/filepath |
| `internal/ontocli` close.go | `onto close` (gate+deps+dirty+archive move) | ontostate, os, cobra |

`onto` imports none of homonto's `internal/{cli,engine,config,adapter,catalog}`.

## Risks / Trade-offs

- **Archive move is not transactional with the Save** → `Save` (atomic) then
  `Rename` (atomic). Between them, a crash leaves `archived:true` in the still-in-
  place change dir; a re-run finds phase close + archived and can complete the
  move. Acceptable for a single-shot CLI; the no-clobber check prevents
  double-archiving into an existing target.
- **DepsResolved is glob-based** → matches the date-prefixed archive convention;
  a dep name that is a prefix of another (e.g. `a` vs `ab`) is disambiguated by
  the `-` separator (`*-a` won't match `…-ab`). Documented + tested.
- **Dirty/undeterminable both block** → conservative for a release-critical move,
  consistent with `onto advance`'s verify→close block.

## Testing Strategy

1. ontostate: `DepsResolved` — resolved+unresolved mix returns the unresolved
   subset; nil/empty → empty; the `a` vs `ab` prefix case.
2. `onto close` (temp git workspace): archives a close-phase change with resolved
   deps + clean tree (dir moved, `archived:true`, exit 0); refuses non-close
   phase; refuses unresolved dep (names it); refuses dirty worktree; refuses when
   the archive target exists (no-clobber). Each refusal asserts the change dir is
   untouched (still under docs/changes/<name>, not moved).
3. Isolation grep; both binaries build; `go test [-race] ./...`, vet, gofmt, tidy.

## Open Questions

None blocking. This completes the onto workflow engine; `onto doctor` (#4) will
add health checks over the resulting `docs/` tree.
