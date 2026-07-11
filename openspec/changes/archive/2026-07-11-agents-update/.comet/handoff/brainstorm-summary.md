# Brainstorm Summary
- Change: agents-update
- Date: 2026-07-11
## Confirmed Technical Approach
v2 #4. `agentsUpdateCmd` (internal/cli/agents.go): setup like add; undeclared→err; non-local→"not yet supported"; not-installed→err→`agents add`; resolve source; per declared target re-materialize by mode — copy: up-to-date if on-disk hash==source hash, else BACKUP dst→dst.bak ONLY when on-disk != prev.Hash AND != source hash (genuine local edit) then WriteAtomic source; link: up-to-date if isSymlinkTo else link.Link; record Install{path,source-hash}; Save lock; print status. Reuses add helpers (isSymlinkTo/link.Link/fsutil.WriteAtomic/subagentpath.Dir/agentlock). Declarative model → NO pin command (version=config). Backup not 3-way-merge (deferred #5).
## Key Trade-offs and Risks
- backup (lossless, simple) vs merge (deferred). One-level .bak (second update overwrites). update refuses uninstalled agent (points to add) — distinct from add.
## Testing Strategy
TDD RED first; build via agents add then perturb (edit source / edit install / both). E2E real binary. Full regression.
## Spec Patches
None. agent-lifecycle ADDED requirement carries update scenarios.
