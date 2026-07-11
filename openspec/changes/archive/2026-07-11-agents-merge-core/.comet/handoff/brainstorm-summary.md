# Brainstorm Summary
- Change: agents-merge-core
- Date: 2026-07-11
## Confirmed Technical Approach
v2 #5a (foundation for 3-way-merge; approved design docs/superpowers/specs/2026-07-11-agents-3way-merge-design.md). `internal/merge.Merge(base,local,upstream)→(result,conflicts)`: line LCS(base,local)+LCS(base,upstream), anchors=base indices in BOTH, sentinels, per-gap 4-way rule (L==B→U / U==B→L / L==U→L / else conflict markers `<<<<<<< local`/`=======`/`>>>>>>> source`). `internal/agentblob`: `.homonto/agents-blobs/<sha256>` Put(idempotent, hash==agentlock.HashContent, WriteAtomic)/Get. add/update Put source content (base for future merge) — behavior-preserving. NO merge-into-update yet (#5b).
## Key Trade-offs and Risks
- anchor-intersection may over-conflict vs full diff3 — always SAFE (never mis-merges); #5b sidecar makes conflicts non-destructive.
- trailing-newline must round-trip exactly (Merge(x,x,x)==x) — explicit test.
- no blob GC (deferred).
## Testing Strategy
TDD RED first; exhaustive merge property tests; blob round-trip/idempotent; blob-persisted-on-add/update + existing tests still green. E2E. Full regression.
## Spec Patches
None. agent-lifecycle ADDED requirements carry merge + blob scenarios.
