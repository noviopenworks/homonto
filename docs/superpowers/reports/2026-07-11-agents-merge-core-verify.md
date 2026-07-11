# Verification Report: agents-merge-core (v2 #5a)

- **Change**: `agents-merge-core` — line-based 3-way merge engine + base-content blob store
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (2 new packages + cli wiring; agent-lifecycle capability)
- **Result**: PASS — final review found no bugs (merge algorithm correct on all axes)

## Scope

New `internal/merge` (pure line diff3), new `internal/agentblob` (content-addressed
`.homonto/agents-blobs/<sha256>`), `internal/cli/agents.go` (`add`/`update` persist
base blobs, behavior-preserving). Foundation for #5b (merge into `update`). Per the
approved `2026-07-11-agents-3way-merge-design.md`.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` (D1 merge algo, D2 blob, D3 wiring) | PASS |
| 3 | Matches Design Doc | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc + approved merge design locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| merge: no changes | `TestMergeRoundTripIdentical` (7 shapes) | PASS |
| merge: only local / only upstream | `TestMergeOnlyLocalChanged` / `...Upstream...` | PASS |
| merge: non-overlapping auto-merge | `TestMergeDisjointEditsAutoMerge`, `...InsertionsAtBothEnds...` | PASS |
| merge: overlapping conflict | `TestMergeOverlappingEditsConflict`, `...Adjacent...` | PASS |
| merge: identical both sides | `TestMergeIdenticalEditBothSides` | PASS |
| merge: deletes / table | `TestMergeTable` (all_empty, deletes, disjoint-no-nl) | PASS |
| blob: install persists retrievable base | cli add/update blob tests | PASS |
| blob: Put idempotent + content-addressed | `internal/agentblob` tests | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 419 passed, 26 packages |
| `go test -race ./internal/merge/... ./internal/agentblob/... ./internal/cli/...` | 64 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |
| `go mod tidy` | no change (no new deps) |

## Independent checks + E2E

- Independent merge spot-checks (my own inputs): disjoint edits (line 1 + line 5)
  → auto-merged to both edits, 0 conflicts; overlapping edits (both change line 2)
  → conflict with all three markers + both contents. Correct.
- E2E (real `homonto`): `agents add` → `.homonto/agents-blobs/<recorded-hash>`
  holds the exact source content; `agents update` to a new source → a second base
  blob appears (both bases retained, so #5b can read `blob(prev.Hash)` as the
  common ancestor). No user-visible behavior change to add/update/doctor.

## Code review (review_mode: standard) — no bugs

The review verified the merge algorithm correct on all seven axes: round-trip
exactness (`Merge(x,x,x)` byte-identical for every newline shape via `SplitAfter`
line-keeps-`\n`); a genuine DP LCS (not greedy) with deterministic tie-break;
anchors = base indices common to both LCSs, strictly increasing in all three
coords, so gap slicing is always in-range (sentinels handle first/last gaps);
correct auto-merge-vs-conflict per gap; **no silent line loss** (every gap takes
one side's literal lines or emits both inside markers — a suboptimal anchor set
can only widen a spurious conflict, never drop/substitute content); blob hash ==
`agentlock.HashContent` (== lockfile key), idempotent Put, Get distinguishes
missing from error, no path-traversal (sha256-hex only); behavior-preserving
wiring. One MINOR (a diff3 universal): an agent line literally equal to a marker
string isn't escaped — standard limitation, no action.

## Conclusion

Verification PASS. Fifth v2 area, foundation slice (#5a): the three-way-merge
engine + base-content blob store, behavior-preserving. #5b wires the merge into
`update` (with the approved `.merged` sidecar conflict UX); #5c adds `update
--all`; #6 remote sources.
