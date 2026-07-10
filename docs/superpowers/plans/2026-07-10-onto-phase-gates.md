---
change: onto-phase-gates
design-doc: docs/superpowers/specs/2026-07-10-onto-phase-gates-design.md
base-ref: 6a41f8a09e819d3977a12fa8dfcf7cdc31791c3d
---

# Onto Phase Gates Implementation Plan

> Implement task-by-task with TDD.

## Global Constraints (Design Doc + delta spec)

- onto binary #3b. Adds per-phase `RequiredArtifacts`, `NextPhase`,
  `TasksAllChecked` to `internal/ontostate`, and `onto advance` to
  `internal/ontocli`.
- Phase order (fixed): open → design → build → verify → close.
- `onto advance`: gate → validChangeName → load → NextPhase → precondition
  (`RequiredArtifacts(next)` all exist; leaving build also needs
  `TasksAllChecked`) → dirty-worktree (WARN normal, BLOCK verify→close) → Save.
  Non-zero + NO phase write on any precondition/close-block failure.
- `onto` stays isolated: imports none of homonto internal/{cli,engine,config,
  adapter,catalog}. Shelling to `git` (VCS) is allowed.
- No new dependency. Gates: build (both binaries), test [-race], vet, gofmt,
  mod tidy clean.

## Task 1: phase helpers in `internal/ontostate`

**Files:** modify `internal/ontostate/state.go`, `internal/ontostate/state_test.go`.

- [x] 1.1 (RED first) Make `RequiredArtifacts(phase)` CUMULATIVE: open→[onto-state.yaml,proposal.md,tasks.md]; design→+design.md; build→+plan.md; verify & close→+verification.md; unknown→open base. Update/extend tests; ensure existing `ValidateSkeleton` tests still pass (adjust any build-phase fixture to include plan.md, or assert the new missing-plan.md error).
- [x] 1.2 (RED first) `NextPhase(phase string) (string, bool)` over ["open","design","build","verify","close"]: successor+true; ("",false) at close & unknown. Tests: each step; close→false; "bogus"→false.
- [x] 1.3 (RED first) `TasksAllChecked(tasksPath string) (bool, error)`: read file; true iff at least one checkbox line and no unchecked one. Tests: all checked→true; one unchecked→false; no checkboxes→false; missing file→error.
- [x] 1.4 GREEN; gofmt/vet clean for internal/ontostate.
- [x] 1.5 Commit: `feat(ontostate): per-phase RequiredArtifacts + NextPhase + TasksAllChecked`

## Task 2: `onto advance` command (`internal/ontocli`)

**Files:** create `internal/ontocli/advance.go`, `internal/ontocli/advance_test.go`; modify `internal/ontocli/root.go` (register).

- [x] 2.1 (RED first) `worktreeDirty(root string) (dirty bool, determinable bool)`: `exec.Command("git","-C",root,"status","--porcelain")`; dirty iff trimmed stdout non-empty; determinable=false on git error / non-repo. Tests via a temp git repo (init in t.TempDir; clean→(false,true); create+don't-commit a file→(true,true)); a non-repo temp dir→(_,false).
- [x] 2.2 (RED first) `advanceCmd()` (ExactArgs(1) `<change>` + `--dir` default "."): gate(root) → validChangeName → Load `<dir>/docs/changes/<name>/onto-state.yaml` (error if missing/invalid) → `NextPhase` (error if !ok) → precondition: stat every `RequiredArtifacts(next)` in changeDir (name first missing); if `st.Phase=="build"` also `TasksAllChecked(<dir>/tasks.md)` (error if false) → dirty check: `d,ok:=worktreeDirty(root)`; if `next=="close"` and (d || !ok) → non-zero (no write); else if d → warn stderr, continue → `st.Phase=next; ontostate.Save(...)`; print `"<change>: <old> → <next>"`, exit 0. Register `advanceCmd()` on the root.
- [x] 2.3 (RED first) Tests via `NewRootCmd().SetArgs([]string{"advance",name,"--dir",tmp})` in a prepared workspace (framework gate satisfied; build the workspace as a temp git repo so dirty is controllable — commit initial state for the "clean" cases). Helper to seed a change at a given phase (write onto-state.yaml + the RequiredArtifacts for that phase). Cases: open→design when design.md present (Load-back phase==design, exit 0); missing design.md → refused, phase stays open; build change with plan.md + an unchecked task → build→verify refused, phase stays build; close change → error, no write; verify change (+verification.md) in a DIRTY repo → verify→close blocked, phase stays verify; open change (+design.md) in a DIRTY repo → advance proceeds (exit 0) with a warning.
- [x] 2.4 GREEN; `grep -E "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go` empty; gofmt/vet clean.
- [x] 2.5 Commit: `feat(onto): 'onto advance' gates phase transitions (+ dirty-worktree block on close)`

## Task 3: regression and docs

- [x] 3.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` then `git diff --exit-code go.mod go.sum` (clean); E2E: in a prepared temp GIT workspace `onto new demo`, add `design.md`, commit, `onto advance demo` → `demo: open → design`, and `onto status` shows phase design.
- [x] 3.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3b (`onto advance` valid-gate transitions + dirty-worktree block on close) landed; remaining onto = dependency resolution + archive/close rules (#3c), `onto doctor` (#4), dual-binary packaging (#5). No over-claim.
- [x] 3.3 Commit all changes.
