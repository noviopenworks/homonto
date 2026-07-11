## 1. `agents update` three-way merge (`internal/cli`)

- [x] 1.1 (TDD RED first) Rewrite `agentsUpdateCmd` copy-mode per Design Doc D1: up-to-date no-op; BASE=`agentblob.Get(prev.Hash)`; missing base/on-disk → fallback backup+overwrite; else `merge.Merge(base,cur,content)` — 0 conflicts → write result (+ `.bak` of prior local when it changes) + advance base (Install.Hash=source hash, `agentblob.Put(source)`); ≥1 conflict → write `<dst>.merged`, leave live dst + that target's lock entry unchanged, exit non-zero. Save merged targets; on any conflict return a non-zero summary error. Link mode unchanged.
- [x] 1.2 (TDD RED first) Tests (build via add, then perturb local + source): disjoint local+source edits → dst has both, no `.merged`, base advanced (doctor healthy after); overlapping edits → live dst UNCHANGED, `<dst>.merged` has markers, exit non-zero, lockfile entry for that target unchanged; idempotent (no perturbation) → "up to date", no `.merged`/`.bak`; missing base blob (delete the blob) + local edit + source change → fallback backs up local to `.bak` and writes source; multi-target where one conflicts → clean target advanced, conflicted target sidecar + non-zero.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update' three-way-merges local edits (safe .merged sidecar)`

## 2. `agents doctor` merge-model reframe (`internal/cli`)

- [x] 2.1 (TDD RED first) Per Design Doc D2: drop the `modified on disk` problem finding; add a `<ti.Path>.merged`-exists → "conflicted" finding; keep the rest. Update the prior #3 doctor tests that asserted modified-on-disk (a locally-edited install is now healthy). Tests: locally-edited install (source unchanged) → doctor NOT a problem (exit 0 absent other issues); a `.merged` sidecar → doctor "conflicted" + non-zero; source-changed + missing-on-disk still findings.
- [x] 2.2 GREEN; gofmt/vet clean. Commit: `feat(cli): agents doctor reframed for merge model (local edits ok, conflicts reported)`

## 3. Regression and docs

- [x] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add; local edit disjoint from a source edit → `agents update` merges both (doctor healthy); overlapping edits → update writes `<dst>.merged`, live file intact, exit non-zero, doctor "conflicted"; resolve (copy .merged over dst, rm .merged) + update → clean.
- [x] 3.2 Update `docs/roadmap.md` v2 status + README (`agents update` now merges; conflicts → `.merged`). No over-claim.
- [x] 3.3 Commit all changes.
