# Tasks — fix-stale-canonical-specs

## 1. Verify source-behavior claims in the deltas (no fabrication)
- [x] Confirm `internal/cli/root.go` registers `version`/`init`/`import`/`plan`/`apply`/`status`/`doctor` and no `agents` group. Verified `root.go:20-28`: `version` is a subcommand; delta lists exactly these seven.
- [x] Confirm `internal/config/config.go` Option C fold matches the config-model delta scenarios. Verified `config.go:509-527`: user scope, builtin+empty→copy, agents-win-over-subagents, `c.Agents=nil`.
- [x] Confirm `internal/agentlock` / `internal/agentblob` and `internal/cli/agents.go` are gone, and check `internal/merge` callers. Verified absent; `internal/merge` has **zero non-test callers**, so removing the "Three-way merge engine" requirement drops no live behavior (the dead package is separate cleanup).

## 2. Add the CI spec↔code correspondence check
- [x] Added `scripts/spec-command-check.sh`: extracts backtick `` `homonto <cmd>` `` command tokens from `openspec/specs/**` and fails on any not registered by the actual binary (`homonto --help`).
- [x] Wired into `scripts/gate.sh` (step "spec<->command correspondence"), so CI + release run it too.
- [x] Proven: fails on the current canonical specs (2 violations: agent-lifecycle, cli-commands `homonto agents`, exit 1); passes on a preview of the post-sync tree (exit 0).

## 3. Verify already-correct docs (no edit expected)
- [x] `README.md:118` and `docs/guides/using-homonto.md:14` already state the `[agents]` fold with "no separate imperative command group." No edit needed.

## 4. Verification gate
- [x] `openspec validate --all --no-color` → 16 passed, 0 failed (delta + 15 specs).
- [x] `go build ./...` and `go vet ./...` green. **Sequencing note:** `scripts/gate.sh` now includes `spec-command-check`, which is red on the *canonical* specs until this change's archive syncs the deltas into `openspec/specs/` — that sync is this change's deliverable. Proven green on the post-sync preview; it flips green on the live tree at archive. Full `gate.sh` is therefore run at/after archive, not mid-build.
- [x] Delta scenarios consistent with source (task 1).

## 5. Out of scope (recorded, do not implement here)
- [x] (note only) `config.go:526` silently discarding `[agents]` after the fold is F35-adjacent — separate change.
- [x] (note only) `docs/superpowers/*` historical residue is F19 — separate change.
