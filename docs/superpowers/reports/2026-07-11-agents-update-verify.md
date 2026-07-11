# Verification Report: agents-update (v2 #4)

- **Change**: `agents-update` — `homonto agents update` (backup-safe re-materialize)
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (agent-lifecycle capability + cli)
- **Result**: PASS — one IMPORTANT data-loss finding fixed during build

## Scope

`internal/cli/agents.go` (`agents update` subcommand), tests. Reuses `add`'s
install helpers + `agentlock`. Backup-before-overwrite; three-way-merge deferred.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 flow, D2 backup semantics) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| re-materialize changed source | `TestAgentsUpdateSourceChangedInstallUntouched` | PASS |
| back up locally-modified install | `TestAgentsUpdateBacksUpLocalEdit` | PASS |
| back up foreign file at new target (added post-review) | `TestAgentsUpdateNewTargetBacksUpForeignFile` | PASS |
| idempotent | `TestAgentsUpdateIsIdempotent` | PASS |
| requires prior install | `TestAgentsUpdateNotInstalled` | PASS |
| builtin not supported | `TestAgentsUpdateBuiltinNotSupported` | PASS |
| undeclared error | `TestAgentsUpdateUndeclared` | PASS |
| link mode | `TestAgentsUpdateLinkModeUpToDate` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 392 passed, 24 packages |
| `go test -race ./internal/cli/...` | 40 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, temp $HOME)

Added a copy agent; edited the source → `agents update` refreshed the install (no
`.bak`, correct — untouched install), and `agents doctor` then reported `healthy`.
Locally edited the install AND the source → `agents update` wrote the new source
and backed up the local edit to `<path>.bak` (verified content). A re-run reported
`up to date`. A newly-declared target with a pre-existing foreign file → `update`
backed the foreign file up to `.bak` (verified) before writing the source — no
silent data loss.

## Code review (review_mode: standard) — one IMPORTANT (FIXED)

The review verified the core backup logic correct (fires only on a genuine local
edit, not on an untouched-but-stale install, not when idempotent, not when
missing; `.bak` written before the overwrite — no data loss on that path) and
found one **IMPORTANT** data-loss gap: the new-target path (`hadRec=false`)
overwrote a pre-existing foreign file with no backup (the case `add` refuses).
**Fixed**: the backup guard now fires whenever the existing on-disk file is not
our own untouched install (`!(hadRec && on-disk == prev.Hash)`), covering both a
local edit and a foreign file at a newly-declared target, with a regression test.
Also confirmed: not-installed/undeclared ordering, lockfile refresh + Save, link
mode (no spurious backup; `link.Link` refuses to clobber a foreign non-symlink).

## Conclusion

Verification PASS. Fourth v2 increment — `agents update` (backup-safe
re-materialize), the fix action for `agents doctor`'s drift. No path overwrites a
user file without a `.bak`. Deferred: three-way-merge, builtin/remote sources,
`migrate`, de-declared-target pruning, per-agent scope.
