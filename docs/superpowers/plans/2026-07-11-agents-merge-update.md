---
change: agents-merge-update
design-doc: docs/superpowers/specs/2026-07-11-agents-merge-update-design.md
base-ref: 20d0e60e2e1cf63c758125235bdf2477a22371c0
---

# Plan: agents merge into update (v2 #5b)

Wire the #5a merge engine into `agents update` (safe `.merged` sidecar);
reframe `doctor`. See Design Doc D1/D2/D3. TDD.

## Task 1: `agents update` 3-way merge
- [x] 1.1 (TDD RED first) Rewrite `agentsUpdateCmd` copy-mode per D1: up-to-date no-op; BASE=agentblob.Get(prev.Hash); missing→fallback backup+overwrite; else merge.Merge(base,cur,source) — 0 conflicts→write result (+.bak of prior local when changed) + advance base (Hash=source, agentblob.Put(source)); ≥1 conflict→write `<dst>.merged`, live dst + lock entry unchanged, non-zero. Save merged targets; any conflict→non-zero summary. Link unchanged.
- [x] 1.2 (TDD RED first) Tests: disjoint→both edits, no .merged, base advanced; overlap→live dst unchanged + .merged markers + non-zero + lock unchanged; idempotent; missing-base fallback→.bak+source; multi-target partial conflict.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update' three-way-merges local edits (safe .merged sidecar)`

## Task 2: `doctor` reframe
- [x] 2.1 (TDD RED first) D2: drop modified-on-disk finding; add `<path>.merged`→"conflicted"; update #3 tests that asserted modified-on-disk. Tests: local-edit (source unchanged)→not a problem (exit 0); .merged→"conflicted"+non-zero; source-changed/missing still findings.
- [x] 2.2 GREEN; gofmt/vet clean. Commit: `feat(cli): agents doctor reframed for merge model`

## Task 3: Regression and docs
- [x] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E: disjoint edits→update merges (doctor healthy); overlap→.merged + live intact + non-zero + doctor conflicted; resolve+update→clean.
- [x] 3.2 Update `docs/roadmap.md` v2 + README (update now merges). No over-claim.
- [x] 3.3 Commit all changes.
