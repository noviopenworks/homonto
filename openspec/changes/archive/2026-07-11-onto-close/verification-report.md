# Verification Report: onto-close (#3c)

- **Change**: `onto-close` — `onto close` archive command + `DepsResolved` dependency resolution
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (scale: 10 tasks, 1 capability, 13 changed files)
- **Result**: PASS — no CRITICAL or IMPORTANT issues

## Scope (source diff vs base `29a4117`)

| File | Change |
|------|--------|
| `internal/ontostate/state.go` | `DepsResolved(root, deps) []string` (+18) |
| `internal/ontostate/state_test.go` | 4 dep-resolution tests (+46) |
| `internal/ontocli/close.go` | `closeCmd()` / `runClose()` (+94, new) |
| `internal/ontocli/close_test.go` | 5 close tests (+204, new) |
| `internal/ontocli/root.go` | register `closeCmd()` (+1) |
| `docs/roadmap.md` | Immediate Next Work: #3c landed |

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md tasks completed `[x]` | PASS (10/10) |
| 2 | Implementation matches `design.md` decisions | PASS |
| 3 | Implementation matches Design Doc (`docs/superpowers/specs/2026-07-10-onto-close-design.md`) | PASS |
| 4 | All capability spec scenarios pass | PASS (see mapping) |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS (no incremental spec changes during build) |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Spec scenario | Test | Result |
|---|---|---|
| resolved/unresolved deps distinguished | `TestDepsResolved_OneArchivedOneMissing_ReturnsMissingOnly` | PASS |
| no deps always resolved (nil/empty) | `TestDepsResolved_NilDeps…`, `…EmptyDeps…` | PASS |
| prefix collision (`a` vs `ab`) | `TestDepsResolved_PrefixCollision_DoesNotResolveShorterDep` | PASS |
| close archives close-phase change | `TestCloseCommand_Success` | PASS |
| refuse non-close phase | `TestCloseCommand_NonClosePhaseRefused` | PASS |
| refuse unresolved dep (names it) | `TestCloseCommand_UnresolvedDepRefused` | PASS |
| blocked by dirty worktree | `TestCloseCommand_DirtyWorktreeRefused` | PASS |
| refuse to clobber existing archive | `TestCloseCommand_ArchiveTargetExistsRefused` | PASS |

Every refusal-path test asserts (a) an error is returned, (b) `docs/changes/<name>/` still exists, and (c) `archived` stays `false` — guarding against false-green.

## Commands run

| Command | Result |
|---|---|
| `go build ./...` (both binaries) | Success |
| `go test ./... -count=1` | 313 passed, 23 packages |
| `go test -race ./...` | 313 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |
| `go mod tidy` + diff go.mod/go.sum | clean |
| isolation `grep -E "internal/(config|engine|adapter|catalog)"` in ontocli/ontostate | empty (onto stays isolated) |

## E2E (temp git workspace)

`onto init` → `onto new demo` → seed design.md/plan.md/verification.md + all-checked tasks.md → commit → `onto advance` open→design→build→verify→close (committing between) → `onto close demo`: change moved to `docs/changes/archive/2026-07-11-demo/`, moved `onto-state.yaml` has `archived: true`, original active dir gone, exit 0. Verified.

## Non-blocking observation (accepted, not a defect)

- `runClose` sets `Archived=true` and `Save`s to the **active** `onto-state.yaml` *before* `os.MkdirAll` + `os.Rename`. If `Rename` failed (e.g. cross-device), the in-place state would read `archived: true` while still active — a partial-state window. This exactly matches the design/task ordering, is guarded by the archive-target pre-existence check, and `os.Rename` is atomic within one filesystem (docs/changes stays on one device). No fix required; recorded for parity with prior OF-style follow-ups.

## Conclusion

Verification PASS. The change is small, isolated, and fully covered; it completes the onto workflow engine (create → advance → close). Ready for branch handling and archive.
