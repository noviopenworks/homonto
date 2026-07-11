# Brainstorm Summary
- Change: agents-update-all
- Date: 2026-07-11
## Confirmed Technical Approach
v2 #5c (last approved merge slice; migrate=update --all). Extract `runAgentUpdate(cmd,name,c,lock,cfgDir,homontoDir,home)→(conflicted,err)` from agentsUpdateCmd body (per-agent 3-way merge, mutates lock.Agents[name], prints, NO Save). Add `--all` bool + ArbitraryArgs: `all&&args>0`→usage err; `!all&&args!=1`→usage err; single→helper+Save+conflicted-nonzero (unchanged); --all→loop sortedKeysAgents(lock): orphan(not in config)→skip note; else helper (err→print+hadError, conflict→anyConflict); Save once; summary; non-zero if anyConflict||hadError. Existing single-update tests MUST stay green (refactor guard).
## Key Trade-offs and Risks
- refactor must preserve #5b single behavior (existing update tests are the guard).
- partial --all: clean advanced, conflicted kept prev, non-zero; re-run idempotent for clean.
- orphan skip (no prune; doctor's concern).
## Testing Strategy
TDD RED first; --all mergeable+up-to-date→exit0; --all with conflict→sidecar+nonzero+other processed; orphan skip; usage errors (name+--all / neither); single update still works.
## Spec Patches
None. agent-lifecycle ADDED requirement carries update --all scenarios.
