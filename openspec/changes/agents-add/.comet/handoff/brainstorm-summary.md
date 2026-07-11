# Brainstorm Summary
- Change: agents-add
- Date: 2026-07-11
## Confirmed Technical Approach
v2 #2 (first lifecycle mutation). New pkg `internal/agentlock` (`.homonto/agents-lock.json`: Lock{Agents map[name]Agent{Source,Version,Mode,Targets,Installed map[tool]{Path,Hash}}}, Load/Save atomic+deterministic, HashContent sha256). `homonto agents add <name>` (internal/cli/agents.go): load config→find agent (undeclared err); local: only (builtin/remote→"not yet supported"); resolve homonto/agents/<x>.md (missing→err); TWO-PASS per agent: conflict-scan all targets (unmanaged file at dst→refuse, install nothing), then install (copy=fsutil.WriteAtomic / link=link.Link) into subagentpath.Dir(tool,"user",home,"")/<name>.md, record Installed; idempotent (copy hash-match/link target-match→no-op); Save lock; print per-target. Reuses subagentpath.Dir/fsutil.WriteAtomic/link.Link.
## Key Trade-offs and Risks
- Lockfile separate from state.json (agent lifecycle ground truth; agents doctor reads it later).
- User scope only (no scope field yet).
- Agents install into same tool agent dir as [subagents] → conflict check refuses to clobber non-owned files (safe); reconciliation deferred.
## Testing Strategy
TDD RED first (agentlock round-trip/determinism; add copy/link/idempotent/conflict/builtin/undeclared/missing-source). E2E real binary. Full regression.
## Spec Patches
None. agent-lifecycle MODIFIED-via-ADDED requirement carries add + lockfile scenarios.
