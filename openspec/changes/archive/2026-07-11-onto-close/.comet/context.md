# Comet Design Handoff

- Change: onto-close
- Phase: design
- Mode: full
- Context hash: e971111ef4837685eb55c9f8c36404431f18d7a8d793495bfc0d7cd6d1e981f9

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-close/proposal.md

- Source: openspec/changes/onto-close/proposal.md
- Lines: 1-58
- SHA256: 78bd619d0687fbe06a4d334db9853b740847ada55404c68c3ba53297d5ceeefd

```md
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

```

## openspec/changes/onto-close/design.md

- Source: openspec/changes/onto-close/design.md
- Lines: 1-88
- SHA256: 7da19fc7588e067d7e257b659a74e0e46bb28af6c3f99180f13662d686b68bee

```md
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

```

## openspec/changes/onto-close/tasks.md

- Source: openspec/changes/onto-close/tasks.md
- Lines: 1-18
- SHA256: 19d7632c55c671ef0c77a30a9ec55c9db6876c607f4b8ad482b3971d69387dbe

```md
## 1. Dependency resolution in `internal/ontostate`

- [ ] 1.1 (TDD, RED first) `DepsResolved(root string, deps []string) []string`: for each dep, resolved iff `filepath.Glob(filepath.Join(root,"docs","changes","archive","*-"+dep))` has ≥1 match; return the unresolved subset in order; nil/empty deps → empty. Tests: mix (archived `a`, missing `b` → `["b"]`); nil → empty; empty → empty; the `a` vs `ab` prefix disambiguation (archive `*-ab` present, dep `a` still unresolved).
- [ ] 1.2 GREEN; gofmt/vet clean for internal/ontostate.
- [ ] 1.3 Commit: `feat(ontostate): DepsResolved dependency-resolution helper`

## 2. `onto close` command (`internal/ontocli`)

- [ ] 2.1 (TDD, RED first) `closeCmd()` (ExactArgs(1) `<change>` + `--dir` default "."): gate(root) → validChangeName → Load `<dir>/docs/changes/<name>/onto-state.yaml` (error if missing/invalid) → if `st.Phase!="close"` non-zero (tell user to `onto advance`), nothing archived → `unresolved := ontostate.DepsResolved(root, st.Deps)`; if non-empty non-zero naming them, nothing archived → `d,ok:=worktreeDirty(root)`; if `d||!ok` non-zero, nothing archived → `archiveDir := <root>/docs/changes/archive/<time.Now().Format("2006-01-02")>-<name>`; if it exists → non-zero, nothing archived → `st.Archived=true; ontostate.Save(<changeDir>/onto-state.yaml, st)`; `os.MkdirAll(<root>/docs/changes/archive)`; `os.Rename(changeDir, archiveDir)`; print `"<change>: archived to <archiveDir>"`, exit 0. Register `closeCmd()` on the root.
- [ ] 2.2 (TDD, RED first) Tests via `NewRootCmd().SetArgs([]string{"close",name,"--dir",tmp})` in a prepared temp GIT workspace (gate satisfied). Helper to seed a close-phase change (onto-state.yaml phase close + the RequiredArtifacts for close incl verification.md), commit for clean cases. Cases: success (close change, no deps, clean → dir moved to docs/changes/archive/<date>-<name>, moved onto-state.yaml has archived:true, exit 0, original docs/changes/<name> gone); non-close phase (seed build) → refused, dir NOT moved; unresolved dep (seed deps:[x] with no archived x) → refused naming x, dir NOT moved; dirty worktree → refused, dir NOT moved; archive target pre-exists (mkdir docs/changes/archive/<date>-<name>) → refused, dir NOT moved.
- [ ] 2.3 GREEN; `grep -E "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go` empty; gofmt/vet clean.
- [ ] 2.4 Commit: `feat(onto): 'onto close' archives a completed change (deps + clean-worktree gated, no-clobber)`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; E2E in a temp git workspace: `onto new demo`, add design.md/plan.md, set tasks.md checked, verification.md, advance through open→design→build→verify→close committing between, then `onto close demo` → change moved under docs/changes/archive/.
- [ ] 3.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3c (`onto close` archive + dependency resolution) landed — the onto workflow engine (create→advance→close) is complete; remaining onto = `onto doctor` (#4) and dual-binary release packaging (#5). No over-claim.
- [ ] 3.3 Commit all changes.

```

## openspec/changes/onto-close/specs/onto-binary/spec.md

- Source: openspec/changes/onto-close/specs/onto-binary/spec.md
- Lines: 1-71
- SHA256: aeeb4de90867e6f93d2691eecee425458c1a67e3aedd266c7056edadafaf7b40

```md
## ADDED Requirements

### Requirement: Dependency resolution

`internal/ontostate.DepsResolved(root string, deps []string) []string` SHALL
return the subset of `deps` that are NOT resolved. A dependency `<dep>` is
resolved iff an archived change directory matching
`docs/changes/archive/*-<dep>` exists under `root`. An empty or nil `deps` SHALL
yield no unresolved dependencies (nil and empty slice are equivalent — both mean
"no dependencies").

#### Scenario: resolved and unresolved dependencies are distinguished

- **GIVEN** a workspace where `docs/changes/archive/2026-07-10-a/` exists but there is no archived `b`
- **WHEN** `DepsResolved(root, ["a","b"])` is called
- **THEN** it returns `["b"]` (a is resolved, b is not)

#### Scenario: no dependencies is always resolved

- **WHEN** `DepsResolved(root, nil)` or `DepsResolved(root, [])` is called
- **THEN** it returns an empty list

### Requirement: onto close archives a completed change

`onto close <change>` SHALL archive a completed change. It SHALL run the
framework-install gate, validate the change name, and require ALL of the
following before archiving (each failing case exits non-zero and archives
NOTHING):

- the change's recorded phase is `close` (a change not yet at `close` is
  rejected with guidance to run `onto advance`);
- every dependency listed in the change's `onto-state.yaml` is resolved
  (`DepsResolved` returns empty); otherwise it names the unresolved dependencies;
- the git worktree is clean (a dirty OR undeterminable worktree blocks the
  archive — this is a release-critical operation).

On success it SHALL set `archived: true` in the change's `onto-state.yaml`, then
move `docs/changes/<change>/` to `docs/changes/archive/<YYYY-MM-DD>-<change>/`
(creating the archive directory if needed), and report the archived path. If the
archive target directory already exists it SHALL refuse (non-zero) and move
nothing.

#### Scenario: close archives a close-phase change with resolved deps and a clean worktree

- **GIVEN** a change at phase `close` with no unresolved deps in a clean git worktree
- **WHEN** `onto close <change>` runs
- **THEN** `docs/changes/<change>/` is moved to `docs/changes/archive/<date>-<change>/`, its `onto-state.yaml` has `archived: true`, and the command reports the archived path, exiting 0

#### Scenario: close refuses a change not at the close phase

- **GIVEN** a change at phase `build`
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero telling the user to `onto advance` to close first, and moves nothing

#### Scenario: close refuses when a dependency is unresolved

- **GIVEN** a close-phase change whose `onto-state.yaml` lists a dep that is not archived
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero naming the unresolved dependency and moves nothing

#### Scenario: close is blocked by a dirty worktree

- **GIVEN** a close-phase change with resolved deps in a workspace with uncommitted changes
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero reporting the dirty worktree and moves nothing

#### Scenario: close refuses to clobber an existing archive entry

- **GIVEN** `docs/changes/archive/<date>-<change>/` already exists
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero and moves nothing

```
