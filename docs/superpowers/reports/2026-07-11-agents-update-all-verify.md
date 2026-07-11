# Verification Report: agents-update-all (v2 #5c)

- **Change**: `agents-update-all` — `homonto agents update --all` (bulk 3-way merge)
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (agent-lifecycle: update --all)
- **Result**: PASS — final review found no correctness bugs (two MINOR notes applied)

## Scope

`internal/cli/agents.go`: extracted `runAgentUpdate` helper (per-agent merge; no
Save); `agentsUpdateCmd` gains `--all` (arg/flag validation + aggregate loop over
all installed agents, orphan skip, per-agent error isolation, single Save,
summary, non-zero on any conflict/error). Refactor preserves single-`update`
behavior.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 helper, D2 flag/loop, D3 error isolation) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc + approved merge design locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| --all merges every installed agent | `TestAgentsUpdateAllMergesEveryInstalled` | PASS |
| --all exits non-zero on any conflict, others processed | `TestAgentsUpdateAllConflictExitsNonZeroOthersProcessed` | PASS |
| --all skips an orphan | `TestAgentsUpdateAllSkipsOrphan` | PASS |
| name + --all / neither → usage error | `TestAgentsUpdateAllUsageErrors` | PASS |
| per-agent error isolated (added post-review) | `TestAgentsUpdateAllPerAgentErrorIsolated` | PASS |
| single update unchanged | `TestAgentsUpdateSingleStillWorks` + full update suite | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 429 passed, 26 packages |
| `go test -race ./internal/cli/...` | passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary)

Two installed copy agents. A disjoint source edit on `a` → `agents update --all`:
`a` "merged", `b` "up to date", summary "2 processed, 0 conflicted, 0 skipped, 0
errored", exit 0 (`a`'s dst reflects the new source). Usage errors: `update a
--all` and `update` (no name/flag) both error, exit 1. A conflicting edit on `b`
→ `update --all`: `a` up-to-date (still processed), `b` CONFLICT with `.merged`,
summary "2 processed, 1 conflicted", exit 1.

## Code review (review_mode: standard) — no correctness bugs

The review confirmed the extracted `runAgentUpdate` is character-identical to the
#5b inline body (same merge branches, `bytes.Equal` guard, backup condition,
lockfile mutation with conflicted targets kept on prev, `agentblob.Put` retained),
the single path still Saves on conflict then exits non-zero, and — the key risk —
`lock.Agents[name]` is assigned ONLY on success (after `agentblob.Put`), so a
mid-loop per-agent error never leaves a half-mutated saved lockfile. Deterministic
sorted iteration; orphan skipped without calling the helper; single Save after the
loop; exit non-zero iff any conflict/error; empty lockfile → exit 0. Two MINOR
notes were **applied**: the summary now counts `errored` agents (was omitted), and
a new test covers the `--all` per-agent hard-error branch (missing source → other
agent still processed, non-zero).

## Conclusion

Verification PASS. #5c completes the approved three-way-merge feature set and the
local-agent lifecycle: `homonto agents add / list / doctor / update / update
--all`, with disjoint auto-merge, safe `.merged` conflict sidecar, and bulk
reconcile. Remaining v2 (deferred): builtin/remote agent sources, `[agents]`-vs-
`[subagents]` reconciliation, de-declared-target pruning, per-agent scope,
compatibility checks, blob GC, `--markers` in-file conflict mode.
