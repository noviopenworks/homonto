# Verification Report: agents-prune (v2 polish)

- **Change**: `agents-prune` — `homonto agents prune` removes orphaned/de-declared installs
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (agent-lifecycle: prune)
- **Result**: PASS — one IMPORTANT data-loss finding fixed during build

## Scope

`internal/cli/agents.go`: new `agentsPruneCmd` (+`--dry-run`) removing homonto-
managed installs for orphan agents and de-declared targets, backing up locally-
modified files, clearing `.merged` sidecars, dropping lockfile records. Reuses
`agentlock`.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 algorithm, D2 safety, D3 lock mutation) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| prune an orphaned agent | `TestAgentsPruneOrphanAgent` | PASS |
| prune a de-declared target (agent kept) | `TestAgentsPruneDeDeclaredTarget` | PASS |
| back up a locally-modified install | `TestAgentsPruneBacksUpLocalEdit` | PASS |
| backup failure keeps the file (data-safety, added post-review) | `TestAgentsPruneBackupFailureKeepsFile` | PASS |
| remove leftover `.merged` sidecar | `TestAgentsPruneRemovesMergedSidecar` | PASS |
| nothing to prune | `TestAgentsPruneNothingToPrune` | PASS |
| dry run changes nothing | `TestAgentsPruneDryRun` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 443 passed, 26 packages |
| `go test -race ./internal/cli/...` | 62 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, full lifecycle)

`agents add rev` → installed; removed `rev` from the config → `agents doctor`
reported "installed but no longer declared (orphan)", exit 1; `agents prune
--dry-run` listed "would remove …" + "pruned orphan agent" but left the file and
lockfile untouched; `agents prune` removed the file + dropped the record; `agents
doctor` → `healthy`, exit 0. The add → doctor → prune → doctor loop is coherent.

## Code review (review_mode: standard) — one IMPORTANT (FIXED)

The review confirmed prune touches only recorded managed paths (no arbitrary/
traversal deletion; `name`/`tool` are map keys/log text, never joined into a
path), `--dry-run` performs zero writes and no Save (lockfile byte-identical),
correct orphan-vs-de-declared classification, correct map-aliasing lockfile
mutation, missing-file no-op + idempotency, and `.merged`-only-of-pruned-target
removal. It found one **IMPORTANT** data-loss bug: `pruneFile` swallowed a `.bak`
backup-write error and removed the locally-edited file anyway (e.g. on ENOSPC),
destroying the edit with no backup — diverging from `update`'s abort-on-backup-
failure. **Fixed**: `pruneFile` now returns whether it is safe to drop the record;
a failed required backup KEEPS the file and its lockfile record and reports
`SKIPPED …`. A regression test (`.bak`-as-a-directory forces the failure) locks it
in.

## Conclusion

Verification PASS. Completes the agent-lifecycle cleanup loop (add → doctor
detects orphans/de-declared → prune removes them, backup-safe). Remaining v2:
**remote** sources (explicit first-release non-goal), plus optional polish
(compatibility checks, per-agent scope, blob GC, `--markers`, `[agents]`-vs-
`[subagents]` reconciliation).
