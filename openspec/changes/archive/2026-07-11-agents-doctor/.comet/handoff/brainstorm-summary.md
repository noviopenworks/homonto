# Brainstorm Summary
- Change: agents-doctor
- Date: 2026-07-11
## Confirmed Technical Approach
v2 #3 (read-only health). `agentsDoctorCmd` in internal/cli/agents.go: config.Load (declared) + agentlock.Load (installed); accumulate findings (sorted): declared-not-installed; local: source drift (HashContent(homonto/agents/<x>.md) != recorded / source missing); per declared target not-installed / missing-on-disk (Lstat) / copy modified-on-disk (ReadFile hash != recorded); installed-target-no-longer-declared; orphan. 0 findings→`healthy`+nil; else print+`fmt.Errorf("...N problem(s)...")`→non-zero (like onto doctor). Reuses agentlock.HashContent (same hash add recorded → no false drift).
## Key Trade-offs and Risks
- source-drift (provider changed→re-add) vs modified-on-disk (installed copy edited→future update/merge) are distinct findings; fix actions deferred.
- link mode: only checks recorded file exists this increment.
## Testing Strategy
TDD RED first; build state by running `agents add` in-test then perturbing (edit source / edit installed / delete file). E2E real binary. Full regression.
## Spec Patches
None. agent-lifecycle ADDED requirement carries doctor scenarios.
