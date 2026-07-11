---
comet_change: onto-close
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-onto-close
status: final
---

# Onto Close — Technical Design

Refinement of `design.md` for `onto-close` (onto binary #3c — final sub-increment
of the onto workflow engine). Adds `onto close` (archive a completed change) and
the `DepsResolved` dependency-resolution helper.

## Context

onto #1/#2/#3a/#3b archived: create (`onto new`), advance (`onto advance`),
inspect (`onto status`). #3c adds the terminal archive action and the
"dependencies resolved" invariant, completing create→advance→close. `onto` stays
isolated from homonto's projection pipeline; git worktree checks reuse
`worktreeDirty` from `onto advance`.

## Goals / Non-Goals

**Goals:** `ontostate.DepsResolved`; `onto close <change>` archiving a
close-phase change gated on resolved deps + clean worktree, no-clobber, setting
`archived:true` then moving into `docs/changes/archive/<date>-<name>/`.

**Non-Goals:** `onto doctor` (#4); packaging (#5); main-spec sync (onto's specs
are the change's own docs/ — no comet-style merge); status listing archived
changes; homonto/isolation changes.

## Decisions

**D1 — `DepsResolved(root, deps) []string` in `ontostate`.** For each dep,
resolved iff `filepath.Glob(filepath.Join(root,"docs","changes","archive",
"*-"+dep))` yields ≥1 match. Return the unresolved subset in input order; nil or
empty `deps` → empty result. The `-` separator disambiguates prefix names
(`*-a` does not match `…-ab`). Subsumes the onto-skeleton OF-s1 note: nil and
empty `Deps` both mean "no dependencies".

**D2 — `closeCmd()`: gate → validate → load → phase → deps → dirty → no-clobber
→ archive.** `onto close <change>` (ExactArgs(1) + `--dir` default "."):
1. `gate(root)`; error → nothing archived.
2. `validChangeName(name)`; error → nothing archived.
3. `changeDir := <root>/docs/changes/<name>`; `st, err := Load(<dir>/onto-state.yaml)`
   (error if missing/invalid).
4. `st.Phase != "close"` → non-zero "at %q; run `onto advance` to reach close";
   nothing archived.
5. `unresolved := DepsResolved(root, st.Deps)`; non-empty → non-zero naming them;
   nothing archived.
6. `d, ok := worktreeDirty(root)`; `d || !ok` → non-zero (dirty/undeterminable
   blocks this release-critical move); nothing archived.
7. `archiveDir := <root>/docs/changes/archive/<time.Now().Format("2006-01-02")>-<name>`;
   `os.Stat(archiveDir)==nil` → non-zero "archive target exists"; nothing archived
   (no-clobber).
8. `st.Archived = true; Save(<changeDir>/onto-state.yaml, st)`;
   `os.MkdirAll(<root>/docs/changes/archive, 0o755)`; `os.Rename(changeDir,
   archiveDir)`. Report `"<change>: archived to <archiveDir>"`, exit 0.

`archived` is set BEFORE the move so the moved state file carries it. `Save`
(atomic) then `Rename` (atomic within a filesystem).

**D3 — Reuse, no duplication.** `closeCmd` reuses `gate`, `validChangeName`,
`worktreeDirty`, `ontostate.Load`/`Save` — no new git or gate logic.

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/ontostate` | `DepsResolved` | os, path/filepath |
| `internal/ontocli` close.go | `onto close` (gate+deps+dirty+archive move) | ontostate, os, cobra |

`onto` imports none of homonto's `internal/{cli,engine,config,adapter,catalog}`.

## Risks / Trade-offs

- **Save-then-Rename not one transaction** → both atomic; a crash between leaves
  `archived:true` in the in-place dir, and a re-run (still phase close, target
  absent) completes the move. No-clobber prevents double-archiving.
- **Glob-based DepsResolved** → matches the date-prefixed archive convention;
  prefix-name safety via the `-` separator (tested).
- **Dirty/undeterminable both block** → conservative, consistent with `onto
  advance`'s verify→close block.

## Testing Strategy

1. ontostate: `DepsResolved` resolved+unresolved mix; nil/empty → empty; `a` vs
   `ab` prefix case.
2. `onto close` (temp git workspace, gate satisfied): success (dir moved to
   archive, `archived:true`, original gone, exit 0); non-close phase refused;
   unresolved dep refused (named); dirty refused; archive-target-exists refused.
   Each refusal asserts `docs/changes/<name>` is untouched (not moved).
3. Isolation grep; both binaries build; `go test [-race] ./...`, vet, gofmt,
   tidy.

## Open Questions

None blocking. Completes the onto workflow engine; `onto doctor` (#4) adds
health checks over the resulting `docs/` tree.
