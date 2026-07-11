# Verification Report: onto-doctor (#4)

- **Change**: `onto-doctor` — `onto doctor` read-only workflow/project health command
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (scale: 7 tasks, 1 capability, 10 changed files)
- **Result**: PASS — no CRITICAL or IMPORTANT issues

## Scope (source diff vs base `57140e3`)

| File | Change |
|------|--------|
| `internal/ontocli/doctor.go` | `doctorCmd()` / `runDoctor()` (+106, new) |
| `internal/ontocli/doctor_test.go` | 8 tests + seed helpers (+212, new) |
| `internal/ontocli/root.go` | register `doctorCmd()` (+1) |
| `docs/roadmap.md` | Immediate Next Work: #4 landed |

No new `ontostate` API; the command assembles existing primitives.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md / plan tasks completed `[x]` | PASS (0 unchecked) |
| 2 | Implementation matches `design.md` decisions (D1 ungated read-only, D2 findings+exit, D3 fixed order, D4 reuse ValidateSkeleton) | PASS |
| 3 | Implementation matches Design Doc (`docs/superpowers/specs/2026-07-11-onto-doctor-design.md`) control flow + finding formats | PASS |
| 4 | All capability spec scenarios pass | PASS (8 scenarios ↔ 8 tests) |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS (no incremental spec changes; no Spec Patches) |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Spec scenario | Test | Result |
|---|---|---|
| healthy → healthy, exit 0 | `TestDoctorCommand_Healthy` | PASS |
| missing docs layout dir | `TestDoctorCommand_MissingDocsDir` | PASS |
| invalid onto-state.yaml | `TestDoctorCommand_InvalidActiveState` | PASS |
| phase not matching artifacts | `TestDoctorCommand_PhaseMissingArtifact` | PASS |
| unresolved dependency | `TestDoctorCommand_UnresolvedDep` | PASS |
| active change marked archived | `TestDoctorCommand_ActiveMarkedArchived` | PASS |
| malformed archive entry | `TestDoctorCommand_ArchiveEntryNotArchived` | PASS |
| read-only, no framework install | `TestDoctorCommand_UngatedReadOnly` | PASS |

Tests #6/#7 were hardened after the build-phase review to pin the exact finding
branch (`"active change marked archived"` / `"not marked archived"`) rather than
a substring that another branch could also emit. Test #8 snapshots the tree
before/after and asserts no path was created (read-only invariant).

## Commands run (verification evidence)

| Command | Result |
|---|---|
| `go build ./...` (both binaries) | Success |
| `go test ./... -count=1` | 321 passed, 23 packages |
| `go test -race ./...` | 321 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |
| `go mod tidy` (during build) | clean |
| isolation grep in ontocli/ontostate | empty (onto stays isolated) |

## E2E (temp git workspace)

Built `onto`; `onto init` + `onto new demo` → `onto doctor` prints `healthy`,
exit 0. Removed `docs/adr` → `docs layout: missing directory docs/adr`, exit 1.
Added an invalid `docs/changes/broken/onto-state.yaml` → two findings, exit 1.
Ran `onto doctor --dir <bare-dir>` (no `homonto.toml`, no docs) → 4 layout
findings, exit 1, and a before/after diff confirmed **no files were created**
(read-only verified end-to-end).

## Code review (review_mode: standard)

One final lightweight review (correctness / security / edge cases) of the whole
change: **no CRITICAL or IMPORTANT findings.** Confirmed short-circuit nil-deref
safety (`err != nil || !info.IsDir()`), archive-excluding glob shape, correct
`continue` short-circuits, correct verdict/exit-code logic, and zero writes.
MINOR notes: loose test assertions (fixed — see above) and the redundant
`ValidateSkeleton` re-load (accepted D4 design trade-off — a diagnostic is not
hot-path and reuse keeps the phase-vs-artifacts check identical to `onto
status`'s).

## Conclusion

Verification PASS. `onto doctor` completes #4 of the onto binary work; only
dual-binary release packaging (#5) remains before `v0.1.0-rc.1`.
