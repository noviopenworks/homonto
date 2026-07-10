## 1. Phase helpers in `internal/ontostate`

- [x] 1.1 (TDD, RED first) Make `RequiredArtifacts(phase)` cumulative per phase: open→[onto-state.yaml,proposal.md,tasks.md]; design→+design.md; build→+plan.md; verify/close→+verification.md; unknown→open base set. Tests: each phase's set; unknown → base. Confirm existing `ValidateSkeleton` tests still pass (a build-phase change missing plan.md now errors).
- [x] 1.2 (TDD, RED first) `NextPhase(phase string) (string, bool)` over ["open","design","build","verify","close"]: successor+true; ("",false) at close and for unknown. Tests: each step, close→false, "bogus"→false.
- [x] 1.3 (TDD, RED first) `TasksAllChecked(tasksPath string) (bool, error)`: read file; true iff ≥1 checkbox and no unchecked `- [ ]`. Tests: all `- [x]` → true; a mix with one `- [ ]` → false; no checkboxes → false; missing file → error.
- [x] 1.4 GREEN; gofmt/vet clean for internal/ontostate.
- [x] 1.5 Commit: `feat(ontostate): per-phase RequiredArtifacts + NextPhase + TasksAllChecked`

## 2. `onto advance` command (`internal/ontocli`)

- [x] 2.1 (TDD, RED first) `worktreeDirty(root string) (dirty bool, determinable bool)` helper: run `git -C <root> status --porcelain` via os/exec; dirty iff non-empty output; determinable=false if git errors / not a repo. Test with a temp git repo (init, clean → not dirty; touch a file → dirty) — use os/exec so the real repo is untouched.
- [x] 2.2 (TDD, RED first) `advanceCmd()` (positional `<change>` ExactArgs(1) + `--dir` default "."): gate(root) → validChangeName → Load `<dir>/docs/changes/<name>/onto-state.yaml` (error if missing/invalid) → `NextPhase` (error if terminal/unknown) → precondition: every `RequiredArtifacts(currentPhase)` file exists (stat loop, name the first missing) AND if leaving build, `TasksAllChecked(tasks.md)` (error if not) → dirty check: if `worktreeDirty` (or undeterminable) and next=="close" → non-zero, no write; else if dirty, warn to stderr and continue → set phase=next, `ontostate.Save`, report `"<change>: <old> → <next>"`, exit 0. Register `advanceCmd()` on the root.
- [x] 2.3 (TDD, RED first) Tests via `NewRootCmd().SetArgs([]string{"advance",name,"--dir",tmp})` in a prepared workspace (gate satisfied; use a temp git repo so dirty state is controllable): open→design when design.md present (Load-back phase==design, exit 0); refuses when design.md missing (phase stays open, non-zero); create a build-phase change (write onto-state.yaml phase build + required files incl plan.md) with an unchecked task → build→verify refused (phase stays build); a change at close → advance errors, no write; a verify-phase change + dirty worktree → verify→close blocked (phase stays verify); a normal advance with a dirty worktree → warns but proceeds.
- [x] 2.4 GREEN; `grep -E "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go` empty; gofmt/vet clean.
- [x] 2.5 Commit: `feat(onto): 'onto advance' gates phase transitions (+ dirty-worktree block on close)`

## 3. Regression and docs

- [x] 3.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; E2E in a prepared temp git workspace: `onto new demo`, add `design.md`, `onto advance demo` → reports `open → design` and `onto status` shows phase design.
- [x] 3.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3b (`onto advance` valid-gate transitions + dirty-worktree block) landed; remaining onto = dependency resolution + archive/close rules (#3c), `onto doctor` (#4), dual-binary packaging (#5). No over-claim.
- [x] 3.3 Commit all changes.
