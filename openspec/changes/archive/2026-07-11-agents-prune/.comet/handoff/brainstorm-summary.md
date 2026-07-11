# Brainstorm Summary
- Change: agents-prune
- Date: 2026-07-11
## Confirmed Technical Approach
v2 polish. `agentsPruneCmd` (prune, NoArgs, --dry-run): config.Load + agentlock.Load; per lockfile agent — orphan(not in config)→prune all recorded target files + drop lock.Agents[name]; else de-declared target(recorded not in TargetsOrAll)→prune that file + drop from Installed. pruneFile: skip if missing; dry-run→"would remove"; else back up to .bak when on-disk hash != recorded base hash, os.Remove file, os.Remove .merged sidecar. Report; "nothing to prune"; --dry-run no writes/Save; Save once when changed. Only recorded managed paths touched. Register under agentsCmd. No blob GC (deferred; content-addressed/shared).
## Key Trade-offs and Risks
- de-declared target: only that target removed, agent + other targets stay.
- no blob GC (separate increment).
- .bak accumulation (one level, deliberate cleanup w/ safety net).
## Testing Strategy
TDD RED first; orphan→file removed+entry gone; de-declared target→that file only+agent stays; local-edit orphan→.bak; .merged removed; nothing-to-prune; --dry-run changes nothing. E2E doctor→prune→doctor healthy.
## Spec Patches
None. agent-lifecycle ADDED requirement carries prune scenarios.
