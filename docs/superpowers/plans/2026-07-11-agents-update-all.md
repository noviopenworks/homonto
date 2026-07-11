---
change: agents-update-all
design-doc: docs/superpowers/specs/2026-07-11-agents-update-all-design.md
base-ref: f1e0661b56619cd899d40cc829c9d356f4b28663
---

# Plan: agents update --all (v2 #5c)

Bulk 3-way-merge over all installed agents (migrate). Refactor per-agent body into
a helper; add `--all`. See Design Doc D1/D2/D3. TDD; existing single-update tests
are the refactor guard.

## Task 1: `agents update --all`
- [ ] 1.1 (TDD RED first) Extract `runAgentUpdate(cmd,name,c,lock,cfgDir,homontoDir,home)→(conflicted,err)` from agentsUpdateCmd (per-agent merge, mutate lock.Agents[name], print, NO Save). Existing single-update tests stay green.
- [ ] 1.2 (TDD RED first) Add `--all` bool + ArbitraryArgs + validation (D2): all&&args>0→usage err; !all&&args!=1→usage err; single→helper+Save+conflicted-nonzero; --all→loop sortedKeysAgents: orphan→skip note; else helper (err→hadError, conflict→anyConflict); Save once; summary; non-zero if anyConflict||hadError.
- [ ] 1.3 (TDD RED first) Tests: --all mergeable+uptodate→exit0+summary; --all with conflict→sidecar+nonzero+other processed; orphan skip→exit0; usage errors (name+--all / neither); single update still works.
- [ ] 1.4 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update --all' bulk-merges every installed agent`

## Task 2: Regression and docs
- [ ] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E: two agents, disjoint source edit on one → `update --all` merges it + other up-to-date + exit0; conflict → .merged + nonzero + other processed.
- [ ] 2.2 Update `docs/roadmap.md` v2 + README (update --all). No over-claim.
- [ ] 2.3 Commit all changes.
