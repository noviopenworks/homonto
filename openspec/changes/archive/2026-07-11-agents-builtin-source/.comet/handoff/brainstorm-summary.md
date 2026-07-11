# Brainstorm Summary
- Change: agents-builtin-source
- Date: 2026-07-11
## Confirmed Technical Approach
v2 #6a. Add `catalog.Catalog.SubagentContent(name)→([]byte,ok,err)` (reads c.fsys at c.subagents[name]; mirrors SubagentPath). Add cli `resolveAgentSource(ag,cfgDir)→([]byte,err)`: local→homonto/agents/<x>.md, builtin→catalog.SubagentContent (unknown→err), else→"remote not yet supported". Wire into add/update(runAgentUpdate)/doctor — replace local-only reads + "not yet supported" branch. builtin has no on-disk path → `builtin:`+`link`→error (copy-only). doctor source-drift uses resolver (builtin drift = catalog upgrade). All lifecycle (hash/materialize/blob/3-way-merge/.merged) source-agnostic → works for builtin free. Bundled agents: code-reviewer/codebase-explorer/comet-navigator. Remote deferred (v1 non-goal).
## Key Trade-offs and Risks
- builtin agent == curated catalog subagent file (coherent; [agents]-vs-[subagents] reconciliation later).
- catalog upgrade = builtin source drift → doctor reports + update merges (desired).
- builtin+link error (minor; copy is the norm).
## Testing Strategy
TDD RED first; catalog SubagentContent known/unknown; resolveAgentSource local/builtin/remote; add builtin (real bundled) installs catalog content; unknown builtin err; builtin+link err; local unchanged (all prior tests green); doctor builtin healthy. E2E real binary w/ builtin:code-reviewer.
## Spec Patches
None. Delta MODIFIES add + update requirements (source resolution).
