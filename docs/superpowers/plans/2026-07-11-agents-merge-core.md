---
change: agents-merge-core
design-doc: docs/superpowers/specs/2026-07-11-agents-merge-core-design.md
base-ref: 8059cbdeb33b1fc984f795815385209e39989ceb
---

# Plan: agents merge core (v2 #5a)

Pure line-based 3-way merge (`internal/merge`) + content-addressed base blob
store (`internal/agentblob`) + add/update persist base blobs (behavior-
preserving). See Design Doc D1/D2/D3. TDD.

## Task 1: `internal/merge`
- [ ] 1.1 (TDD RED first) `Merge(base,local,upstream)→(result,conflicts)` per D1 (line LCS, anchor intersection, sentinels, 4-way gap rule, git markers).
- [ ] 1.2 (TDD RED first) Exhaustive tests (D1 list): identity, one-side, disjoint auto-merge, overlap conflict+markers, identical-both, adjacent, empty, no-trailing-newline round-trip.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(merge): pure line-based three-way merge engine`

## Task 2: `internal/agentblob`
- [ ] 2.1 (TDD RED first) `Put`(idempotent, hash==agentlock.HashContent, WriteAtomic)/`Get` at `.homonto/agents-blobs/<hash>`. Tests: round-trip, idempotent, missing→(nil,false,nil), hash match.
- [ ] 2.2 GREEN; gofmt/vet clean. Commit: `feat(agentblob): content-addressed base-content blob store`

## Task 3: persist base blobs in add/update
- [ ] 3.1 (TDD RED first) add/update `agentblob.Put(homontoDir, sourceContent)` (once/agent; propagate err); no output/flow change. Tests: blob exists+Get after add; new-source blob after update; existing add/update/doctor tests still pass.
- [ ] 3.2 GREEN; gofmt/vet clean. Commit: `feat(cli): persist installed base content to the agent blob store`

## Task 4: Regression and docs
- [ ] 4.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E: `agents add`→blob holds source; no behavior change.
- [ ] 4.2 Update `docs/roadmap.md` v2 status (merge engine + blob store landed; #5b next). No over-claim.
- [ ] 4.3 Commit all changes.
