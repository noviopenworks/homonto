# Brainstorm Summary
- Change: agents-merge-update
- Date: 2026-07-11
## Confirmed Technical Approach
v2 #5b: `update` copy-mode → 3-way merge. Per target: up-to-date no-op; BASE=agentblob.Get(prev.Hash); missing base/on-disk→fallback backup+overwrite; else merge.Merge(base,cur,source) — 0 conflicts→write result (+.bak of prior local when it changes) + ADVANCE base to upstream (Install.Hash=source hash, agentblob.Put(source)); ≥1 conflict→write `<dst>.merged`, leave live dst + that target's lock entry unchanged, exit non-zero (Save merged targets). doctor: DROP modified-on-disk finding (local edits normal now), ADD `<path>.merged`-exists→"conflicted"; keep source-changed/missing/orphan/etc. CRUX: Install.Hash = BASE (ancestor=source last installed/merged-against); on-disk!=base = expected local edits, not drift.
## Key Trade-offs and Risks
- doctor contract change (modified-on-disk dropped) — update #3 tests; delta spec MODIFIES doctor req.
- conflict advances clean targets, leaves conflicted target on prev; re-run after resolve is idempotent.
- stale .merged lingers until user deletes (intended nudge).
## Testing Strategy
TDD RED first; disjoint→merge, overlap→sidecar+non-zero+live-untouched, idempotent, missing-base fallback, multi-target partial conflict; doctor local-edit-ok + conflicted. E2E incl resolve loop. Full regression.
## Spec Patches
None. Delta MODIFIES the update + doctor requirements (both already in main spec from #4/#3).
