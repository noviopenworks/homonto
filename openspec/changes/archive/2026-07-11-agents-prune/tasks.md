## 1. `homonto agents prune` (`internal/cli`)

- [x] 1.1 (TDD RED first) `agentsPruneCmd` (`prune`, NoArgs, `--dry-run` bool) per Design Doc D1/D2/D3: load config + lockfile; for each lockfile agent — orphan (not declared) → prune every recorded target file + drop `lock.Agents[name]`; else de-declared target (recorded target not in TargetsOrAll) → prune that file + drop from Installed. `pruneFile`: skip if missing; dry-run→record "would remove"; else back up to `.bak` when on-disk hash != recorded base hash, remove file, remove `.merged` sidecar. Report actions; `nothing to prune` when none; `--dry-run` changes nothing (no Save). Save once when changed. Register `prune` under `agentsCmd()`.
- [x] 1.2 (TDD RED first) Tests (build via `agents add`, then de-declare in a new config): orphan agent → its file(s) removed + lockfile entry gone; de-declared target (agent keeps another target) → only that target's file removed + Installed entry gone, agent stays; local-edit orphan → `.bak` created with the edit before removal; `.merged` sidecar of a pruned target removed; nothing-to-prune → message, no changes; `--dry-run` → lists prunable but removes nothing and lockfile unchanged. Assert on disk + lockfile.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents prune' removes orphaned/de-declared agent installs`

## 2. Regression and docs

- [x] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add an agent; remove it from config; `agents doctor` reports orphan; `agents prune --dry-run` lists it; `agents prune` removes the file + lockfile entry; `agents doctor` → healthy. A de-declared target similarly pruned.
- [x] 2.2 Update `docs/roadmap.md` v2 status (agents prune landed) + README (mention `agents prune`). No over-claim.
- [x] 2.3 Commit all changes.
