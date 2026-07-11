# Verification Report: agents-merge-update (v2 #5b)

- **Change**: `agents-merge-update` — `agents update` three-way merge + doctor reframe
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (agent-lifecycle: update + doctor MODIFIED)
- **Result**: PASS — final review found no CRITICAL/IMPORTANT bugs

## Scope

`internal/cli/agents.go`: `agentsUpdateCmd` copy-mode rewritten to three-way merge
(base from blob, local on-disk, upstream source), safe `<dst>.merged` sidecar on
conflict, base advances to source on clean merge; `agentsDoctorCmd` reframed
(local edits ok; `.merged` → "conflicted"). Uses #5a `internal/merge` +
`internal/agentblob`.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 merge, D2 doctor, D3 base semantics) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass (update + doctor MODIFIED) | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc + approved merge design locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| non-overlapping edits auto-merge | `TestAgentsUpdateDisjointEditsAutoMerge` | PASS |
| overlapping edits → sidecar + non-zero, live untouched | `TestAgentsUpdateOverlappingConflictSidecar` | PASS |
| update idempotent | `TestAgentsUpdateIsIdempotent` | PASS |
| missing base blob → fallback backup | `TestAgentsUpdateMissingBaseFallsBackToBackup` | PASS |
| foreign file at new target backed up (safety preserved) | `TestAgentsUpdateNewTargetBacksUpForeignFile` | PASS |
| clean-merge backup + base advance | `TestAgentsUpdateBacksUpLocalEdit`, `...PersistsNewBaseBlob` | PASS |
| doctor: local edit not a problem | `TestAgentsDoctorLocalEditIsHealthy` | PASS |
| doctor: pending conflict reported | `TestAgentsDoctorReportsPendingConflict` | PASS |
| doctor: source-changed / missing / orphan intact | existing doctor tests | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 423 passed, 26 packages |
| `go test -race ./internal/cli/... ./internal/merge/...` | passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, full merge lifecycle)

Installed a 6-line copy agent; locally edited line 1 and changed the source line 6
(disjoint) → `agents update` **auto-merged both** (dst has LOCAL1 + UP6), no
`.merged`, and `agents doctor` → `healthy`. Then locally edited line 3 and changed
the source line 3 differently (overlap) → `agents update` reported CONFLICT, exit
1, the **live dst stayed as the local edit (LOCAL3)**, and `<dst>.merged` held the
conflict markers; `agents doctor` reported both "conflicted (resolve …merged)" and
"source changed" (2 findings, exit 1 — coherent: a conflict does not advance the
base). No user content was ever lost or silently overwritten.

## Code review (review_mode: standard) — no CRITICAL/IMPORTANT bugs

The review verified data safety on all eight axes: (1) conflict → only
`<dst>.merged` written, live `dst` untouched, conflicted target keeps its prior
lockfile record, exit non-zero; (2) clean merge records the UPSTREAM (source) hash
+ `agentblob.Put(source)` — base advances to the pristine source, NO double-apply
of edits; (3) backup-before-overwrite on a changing clean merge; (4) missing-base
fallback guarded (`agentblob.Get` never called with an empty hash; foreign file at
a new target is backed up, never silently clobbered — the #4 safety guard was
preserved); (5) multi-target partial conflict advances clean targets, keeps the
conflicted one, still exits non-zero; (6) doctor reframe correct + deterministic;
(7) idempotent; (8) marker/newline round-trip exact. Accepted cosmetic MINOR:
`firstRecordedHash` assumes all targets share a hash — after a partial conflict
they diverge, so the doctor "source changed" line may be spuriously present/absent,
but a `.merged` always keeps doctor non-zero (never a false "healthy"). No fix
required.

## Conclusion

Verification PASS. #5b delivers the three-way-merge payoff: `agents update`
reconciles local edits with upstream changes (auto-merge, safe `.merged` sidecar
on conflict), and `doctor` reflects the merge model. Deferred: #5c `agents update
--all`, `--markers` in-file mode, blob GC, builtin/remote sources.
