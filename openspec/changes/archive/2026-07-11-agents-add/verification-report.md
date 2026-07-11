# Verification Report: agents-add (v2 #2)

- **Change**: `agents-add` — `homonto agents add` (local install) + `.homonto/agents-lock.json`
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (agent-lifecycle capability + new pkg + cli)
- **Result**: PASS — one IMPORTANT security finding fixed during build; MINOR fails-safe accepted

## Scope

New `internal/agentlock` (lockfile model + Load/Save/HashContent),
`internal/cli/agents.go` (`agents add` subcommand), `internal/config` (local:
source traversal guard added post-review), tests, README + roadmap. First
agent-lifecycle mutation.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 lockfile, D2 two-pass install, D3 managed check) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| add local copy-mode agent | `TestAgentsAddCopyInstallsAndRecords` | PASS |
| add is idempotent | `TestAgentsAddIsIdempotent` | PASS |
| add refuses to clobber unmanaged file (all-or-nothing) | `TestAgentsAddConflictIsAllOrNothing` | PASS |
| builtin not yet supported | `TestAgentsAddBuiltinNotSupported` | PASS |
| undeclared agent error | `TestAgentsAddUndeclared` | PASS |
| missing source file error | `TestAgentsAddMissingSourceFile` | PASS |
| link mode symlinks | `TestAgentsAddLinkModeSymlinks` | PASS |
| lockfile round-trip / deterministic / hash | `internal/agentlock` tests | PASS |
| traversal local source rejected (security, added post-review) | `TestAgentsRejectTraversalLocalSource` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 377 passed, 24 packages |
| `go test -race ./internal/agentlock/... ./internal/cli/... ./internal/config/...` | 76 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, temp $HOME)

`[agents.rev] source="local:rev" version="1.0.0" mode="copy" targets=[claude,opencode]`
+ `homonto/agents/rev.md` → `homonto agents add rev` installed
`.claude/agents/rev.md` and `.config/opencode/agent/rev.md`, and wrote
`.homonto/agents-lock.json` recording source/version/mode/targets + each target's
path and content hash. A re-run reported "up to date" for both (idempotent, no
rewrite). A `builtin:` agent → "only local: sources are supported yet" (deferred).
A `source="local:../../secret"` config → **rejected at load** ("must be a plain
name") — the traversal fix, confirmed exit 1.

## Code review (review_mode: standard) — one IMPORTANT (FIXED), one MINOR (accepted)

- **IMPORTANT (fixed):** config-driven path traversal — a `local:` source with
  path components (`local:../../secret`) resolved outside `homonto/agents/` and
  would copy/symlink an arbitrary file on `agents add`. Fixed by validating the
  `local:` source name as a plain name at config load (`validateAgents`), matching
  the agent-name guard, with a regression test.
- **MINOR (accepted):** a mid-operation *non-conflict* I/O failure (e.g. disk
  full on target B after target A wrote) leaves target A's file on disk but
  unrecorded in the lockfile; the next run then refuses (sees it as unmanaged)
  rather than clobbering. This fails *safe* (no corruption, no clobber) — recovery
  friction only. Acceptable for this increment; a later increment can add
  rollback.
- Confirmed correct: all-or-nothing conflict scan (Pass 1 before any write, Save
  once at end), clobber safety (`os.Lstat` + lockfile-path match; a
  `[subagents]`/hand-written file, dangling symlink, or directory at dst is
  refused), idempotency guards (copy hash-match / link target-match skip the
  write), dst-side safety (validated name), link.Link foreign-file refusal,
  lockfile integrity (parse errors surfaced, empty-on-absence).

## Conclusion

Verification PASS. Second v2 increment — the first agent-lifecycle mutation
(`agents add`) + the `.homonto/agents-lock.json` installed-state ground truth.
Deferred: builtin/remote sources, update/pin/doctor/migrate, three-way-merge, a
per-agent scope, and `[agents]`-vs-`[subagents]` reconciliation.
