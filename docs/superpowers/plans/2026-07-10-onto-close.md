---
change: onto-close
design-doc: docs/superpowers/specs/2026-07-10-onto-close-design.md
base-ref: a3137d2a7b2b603412c28e7beba51a3692a24c63
---

# Onto Close Implementation Plan

> Implement task-by-task with TDD.

## Global Constraints (Design Doc + delta spec)

- onto binary #3c (final workflow-engine increment). Adds `ontostate.DepsResolved`
  and `onto close` (archive a completed change).
- `onto close`: gate → validChangeName → load → phase==close → DepsResolved
  (unresolved → refuse) → worktreeDirty (dirty OR undeterminable → refuse) →
  no-clobber archive-target check → set archived:true + Save → os.Rename move.
  ANY failure exits non-zero and moves/archives NOTHING.
- Archive convention: `docs/changes/archive/<YYYY-MM-DD>-<name>/`.
- `onto` stays isolated: imports none of homonto internal/{cli,engine,config,
  adapter,catalog}. Reuse gate/validChangeName/worktreeDirty (already in ontocli).
- No new dependency. Gates: build (both binaries), test [-race], vet, gofmt,
  mod tidy clean.

## Task 1: `DepsResolved` in `internal/ontostate`

**Files:** modify `internal/ontostate/state.go`, `internal/ontostate/state_test.go`.

- [x] 1.1 (RED first) `DepsResolved(root string, deps []string) []string`: for each dep, resolved iff `filepath.Glob(filepath.Join(root,"docs","changes","archive","*-"+dep))` has ≥1 match; return the unresolved subset in input order; nil/empty deps → empty slice. Tests: archived `a` + missing `b` (create `docs/changes/archive/2026-07-10-a/`) → `["b"]`; nil → empty; empty → empty; prefix case (archive `*-ab` present, dep `a`) → `a` unresolved (the `-` separator prevents `*-a` matching `…-ab`).
- [x] 1.2 GREEN; gofmt/vet clean for internal/ontostate.
- [x] 1.3 Commit: `feat(ontostate): DepsResolved dependency-resolution helper`

## Task 2: `onto close` command (`internal/ontocli`)

**Files:** create `internal/ontocli/close.go`, `internal/ontocli/close_test.go`; modify `internal/ontocli/root.go` (register).

- [x] 2.1 (RED first) `closeCmd()` (ExactArgs(1) `<change>` + `--dir` default "."): gate(dir) → validChangeName → `changeDir := filepath.Join(dir,"docs","changes",name)`; `st,err := ontostate.Load(filepath.Join(changeDir,"onto-state.yaml"))` (error if missing/invalid) → if `st.Phase != "close"` return non-zero (tell user to `onto advance` to close), nothing archived → `unresolved := ontostate.DepsResolved(dir, st.Deps)`; if `len(unresolved)>0` return non-zero naming them, nothing archived → `d,ok := worktreeDirty(dir)`; if `d || !ok` return non-zero (dirty/undeterminable blocks close), nothing archived → `archiveDir := filepath.Join(dir,"docs","changes","archive", time.Now().Format("2006-01-02")+"-"+name)`; if `os.Stat(archiveDir)` succeeds return non-zero ("archive target exists"), nothing archived → `st.Archived=true; ontostate.Save(filepath.Join(changeDir,"onto-state.yaml"), st)`; `os.MkdirAll(filepath.Join(dir,"docs","changes","archive"),0o755)`; `os.Rename(changeDir, archiveDir)`; print `fmt.Fprintf(cmd.OutOrStdout(),"%s: archived to %s\n",name,archiveDir)`; return nil. Register `closeCmd()` on the root.
- [x] 2.2 (RED first) Tests via `NewRootCmd().SetArgs([]string{"close",name,"--dir",tmp})` in a prepared temp GIT workspace (gate satisfied; commit initial state for clean cases). Helper `seedClose(t,root,name,deps...)`: MkdirAll `docs/changes/<name>`; `ontostate.Save` onto-state.yaml with phase="close", Deps=deps, + touch RequiredArtifacts("close") (proposal, tasks, design.md, plan.md, verification.md); commit. Cases:
  - success: seedClose(demo, no deps), clean → `close demo` → `docs/changes/archive/<date>-demo/` exists, its onto-state.yaml Loads with Archived==true and Phase=="close", `docs/changes/demo` is GONE, exit 0.
  - non-close phase: seed a change with phase "build" → `close` refused (err!=nil), `docs/changes/<name>` still present.
  - unresolved dep: seedClose(demo, deps=["missing"]) → refused naming "missing", not moved.
  - dirty worktree: seedClose(demo) then write an uncommitted file → refused, not moved.
  - archive-target exists: seedClose(demo), pre-create `docs/changes/archive/<today>-demo` → refused, not moved.
- [x] 2.3 GREEN; `grep -E "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go` empty; gofmt/vet clean.
- [x] 2.4 Commit: `feat(onto): 'onto close' archives a completed change (deps + clean-worktree gated, no-clobber)`

## Task 3: regression and docs

- [x] 3.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` then `git diff --exit-code go.mod go.sum` (clean); E2E in a temp git workspace: `onto new demo`; add design.md, plan.md, verification.md; make tasks.md all-checked; commit; `onto advance` open→design→build→verify→close (committing between); then `onto close demo` → `docs/changes/demo` moved under `docs/changes/archive/`.
- [x] 3.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3c (`onto close` + dependency resolution) landed — the onto workflow engine (create→advance→close) is COMPLETE; remaining onto = `onto doctor` (#4) and dual-binary release packaging (#5). No over-claim.
- [x] 3.3 Commit all changes.
