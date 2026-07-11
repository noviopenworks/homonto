# Verification Report: agents-doctor (v2 #3)

- **Change**: `agents-doctor` — read-only `homonto agents doctor` (declared-vs-installed drift)
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (agent-lifecycle capability + cli)
- **Result**: PASS — final review found no bugs

## Scope

`internal/cli/agents.go` (`agents doctor` subcommand), tests. Read-only; reuses
`internal/agentlock` (`Load`, `HashContent`). Findings + non-zero exit like `onto
doctor`.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 checks, D2 source-hash, D3 determinism) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| healthy | `TestAgentsDoctorHealthy` | PASS |
| declared but not installed | `TestAgentsDoctorDeclaredNotInstalled` | PASS |
| orphaned install | `TestAgentsDoctorOrphan` | PASS |
| source drift | `TestAgentsDoctorSourceDrift` | PASS |
| modified on disk | `TestAgentsDoctorModifiedOnDisk` | PASS |
| missing on disk | `TestAgentsDoctorMissingOnDisk` | PASS |
| read-only | `TestAgentsDoctorIsReadOnly` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 384 passed, 24 packages |
| `go test -race ./internal/cli/...` | 29 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, temp $HOME)

Added a `local:` copy agent, then: `agents doctor` → `healthy`, exit 0; edited the
source file → `source changed since install (re-run …)`, exit 1; deleted an
installed target file → `source changed` + `installed file missing: <path>`,
non-zero exit. Read-only (no files written).

## Code review (review_mode: standard) — no bugs

The final review verified all eight axes correct: read-only (no write/Save/mkdir;
`agentlock.Load` never creates the file); verdict/exit (no false-healthy — every
detected problem appends a finding and the error is returned whenever findings is
non-empty); determinism (every map iteration sorted); source-drift no-false-
positive (`add` records the same hash for every target, so comparing the first
sorted recorded hash is consistent; empty `Installed` safely skips); missing-vs-
modified precedence (`Lstat` fail → finding + continue, no double-finding); copy-
vs-link (hash check gated to copy; link is presence-only). One MINOR, documented
deferred non-goal: a link whose symlink is replaced by a divergent regular file
isn't flagged (though an edited/deleted `local:` **source** is still caught by the
source-drift branch regardless of mode). No fixes required.

## Conclusion

Verification PASS. Third v2 increment — the read-only agent health check the
`update`/`migrate` increments will act on. Deferred: builtin/remote sources,
update/pin/migrate, three-way-merge, per-agent scope, link-target divergence
detection, `[agents]`-vs-`[subagents]` reconciliation.
