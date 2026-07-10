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
