## 1. `agents update --all` (`internal/cli`)

- [x] 1.1 (TDD RED first) Refactor per Design Doc D1: extract `runAgentUpdate(cmd, name, c, lock, cfgDir, homontoDir, home) (conflicted bool, err error)` from the current `agentsUpdateCmd` body (does the per-agent merge, mutates lock.Agents[name], prints per-target statuses, does NOT Save). The existing single-update tests must still pass unchanged.
- [x] 1.2 (TDD RED first) Add `--all` bool flag + `cobra.ArbitraryArgs` + validation (D2): `all && args>0` → usage err; `!all && args!=1` → usage err; single path calls the helper then Save then conflicted→non-zero (unchanged); `--all` path loops `sortedKeysAgents(lock.Agents)` — orphan (not in config)→skip note; else helper (err→print+hadError, else anyConflict); Save once; print summary; return non-zero if anyConflict||hadError.
- [x] 1.3 (TDD RED first) Tests: `update --all` with one disjoint-mergeable + one up-to-date agent → both processed (first merged, second up-to-date), summary, exit 0; one conflicting agent → its `.merged` written + exit non-zero, other still processed; orphan (in lock, not config) → skipped note, exit 0 (absent other issues); `update <name> --all` and `update` (no name, no --all) → usage errors; single `update <name>` still works (all prior update tests green).
- [x] 1.4 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update --all' bulk-merges every installed agent`

## 2. Regression and docs

- [x] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): two installed agents, edit one's source disjointly → `agents update --all` merges it, reports the other up-to-date, exit 0; make one conflict → `update --all` writes its `.merged`, exit non-zero, other processed.
- [x] 2.2 Update `docs/roadmap.md` v2 status (update --all landed; migrate = update --all) + README (mention `agents update --all`). No over-claim.
- [x] 2.3 Commit all changes.
