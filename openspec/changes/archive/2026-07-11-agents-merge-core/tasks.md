## 1. `internal/merge` — line-based three-way merge

- [x] 1.1 (TDD RED first) `merge.Merge(base, local, upstream []byte) (result []byte, conflicts int)` per Design Doc D1: line split w/ exact trailing-newline round-trip; `lcsLineIndices` DP LCS; anchor intersection + sentinels; per-gap 4-way rule; git-style markers (`<<<<<<< local` / `=======` / `>>>>>>> source`).
- [x] 1.2 (TDD RED first) Exhaustive tests: `Merge(x,x,x)==x`&0; `Merge(b,l,b)==l`&0; `Merge(b,b,u)==u`&0; disjoint edits auto-merge&0; overlapping→conflicts≥1 + both markers present; identical edits both sides→0 & single copy; adjacent edits; empty/one-line inputs; no-trailing-newline round-trip; conflict block byte-shape.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(merge): pure line-based three-way merge engine`

## 2. `internal/agentblob` — content-addressed base store

- [x] 2.1 (TDD RED first) `Put(homontoDir, content) (hash, err)` (idempotent, sha256hex == agentlock.HashContent, WriteAtomic) + `Get(homontoDir, hash) (content, ok, err)` at `.homonto/agents-blobs/<hash>`. Tests: Put→Get round-trip; Put idempotent (same hash, single file); Get missing→(nil,false,nil); hash matches agentlock.HashContent.
- [x] 2.2 GREEN; gofmt/vet clean. Commit: `feat(agentblob): content-addressed base-content blob store`

## 3. Persist base blobs in add/update (behavior-preserving)

- [x] 3.1 (TDD RED first) In `agentsAddCmd`/`agentsUpdateCmd`, after materializing, `agentblob.Put(homontoDir, sourceContent)` (once per agent; propagate error). No output/flow change. Tests: after `agents add`, `.homonto/agents-blobs/<recorded hash>` exists and Get returns the source content; after `agents update` to a new source, the new source's blob exists; existing add/update/doctor tests still pass (behavior unchanged).
- [x] 3.2 GREEN; gofmt/vet clean. Commit: `feat(cli): persist installed base content to the agent blob store`

## 4. Regression and docs

- [x] 4.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): `agents add` a local agent → `.homonto/agents-blobs/<hash>` holds the source content; no user-visible behavior change to add/update/doctor.
- [x] 4.2 Update `docs/roadmap.md` v2 status (merge engine + base blob store landed; #5b wires merge into update next). No over-claim (merge not yet wired into update).
- [x] 4.3 Commit all changes.
