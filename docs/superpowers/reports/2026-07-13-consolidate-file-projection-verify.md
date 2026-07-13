# Verification — consolidate-file-projection (F40 file-projection slice)

Full verification (full workflow + delta spec). PASS.

## Checks
| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Implementation matches design.md/Design Doc (type-agnostic []fileproj.Link, no deletes, copy-mode out of scope) | PASS |
| 3 | Delta spec scenarios (adapters plan/apply/observe symlinks via core; core plans no deletes; fail-fast conflict ordering preserved) | PASS |
| 4 | proposal.md goals met (six inline link blocks → one shared core) | PASS |
| 5 | Full suite `go test ./... -race` | PASS (650) |
| 6 | Adapter suites `-race` | PASS (142) |
| 7 | Conformance suite (drift, malformed, secret non-leak, foreign-content) all 3 adapters | PASS (20) |
| 8 | `go vet`, `go build`, `openspec validate --all` | PASS |
| 9 | Code review (standard, manual — security-sensitive symlink diff) | PASS |

## Code review (standard, manual)
Reviewed the delegated migration in full:
- **Apply phase ordering preserved verbatim** (both adapters): ApplyState×3 (adopt/delete state) → Conflicts×3 (fail-fast link precheck) → copy conflict precheck → gated doc writes → ApplyLinks×3 (inactive-prune + link + record) → applyCopySubagents. All conflicts detected before any write/link creation — no partial-write-then-error path.
- **Copy-mode excluded**: subagentFileLinks skips `e.Mode == "copy"`; subagentcopy.*/planCopyOps/applyCopySubagents untouched.
- **No double-delete**: fileproj plans no deletes; generic delete loops unchanged (`filePrefix` claude / `managedPrefix` opencode) remain the single file-prefix delete source. opencode `plugin.*` array membership fully preserved.
- **Identity subtleties honored**: Inactive `""` sentinel (never Join("",name)); `filepath.Base(dst)` unification (skills directory canary green); relink asymmetry (New=bare src vs "dst -> src"); `" -> "` hash separator centralized in fileproj; Observe reads recorded dst (pending scope switch). Dead `recordedDst` removed from both util.go.
- No `_test.go` modified; internal/link + internal/copyfile untouched.

## Behavior / risk
Pure refactor, no behavior/schema change. Six near-identical inline link blocks removed in favor of the shared fileproj core + ~12-line-per-type builders (net -386 lines). Combined with the structured-doc slice, the two adapters dropped claude 1037→762, opencode 999→776. Copy-mode consolidation is a documented follow-on.
