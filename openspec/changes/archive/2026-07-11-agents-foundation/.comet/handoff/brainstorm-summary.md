# Brainstorm Summary
- Change: agents-foundation
- Date: 2026-07-11
## Confirmed Technical Approach
v2 foundation (read-only). `Agent{Source,Version,Targets,Mode}` + `Config.Agents`; validation reuses `validSource`(builtin:/local:)/`validateKey`/target-check; `mode ∈ {"",copy,link}`. Helpers `TargetsOrAll()`/`ModeOrDefault()`. New `internal/cli/agents.go`: `agentsCmd()` parent + read-only `list` subcommand (config.Load, sorted, print name/source/version-or-unpinned/targets/mode). Register on root. NO projection/lockfile/mutation (deferred). Independent of v1 `[subagents]`.
## Key Trade-offs and Risks
- [agents] vs [subagents] overlap: v2 lifecycle model vs v1 symlink Resource; coexist independently this increment; supersession is a later decision.
- list shows declared intent, not installed state (no lockfile yet).
## Testing Strategy
TDD RED first (config model+validation; cli list). E2E: real binary agents list. Full regression.
## Spec Patches
None. New capability agent-lifecycle + config-model ADDED requirement carry the model + list scenarios.
