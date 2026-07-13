# Tasks — fix-stale-canonical-specs

## 1. Verify source-behavior claims in the deltas (no fabrication)
- [ ] Confirm `internal/cli/root.go` registers exactly `version(?)/init/import/plan/apply/status/doctor` and no `agents` group; note whether `version` is a subcommand or `--version` flag and make the cli-commands delta match reality.
- [ ] Confirm `internal/config/config.go` Option C fold matches the config-model delta scenarios (user scope, builtin+empty→copy, agents-win-over-subagents, `c.Agents=nil`). Adjust the delta if source differs.
- [ ] Confirm `internal/agentlock` / `internal/agentblob` are absent and `internal/cli/agents.go` is gone; grep for any live caller of `internal/merge` and record which capability (if any) still owns it, so the agent-lifecycle removal drops no real behavior.

## 2. Add the CI spec↔code correspondence check
- [ ] Add a script (e.g. `scripts/spec-command-check.sh` or `.mjs`) that extracts every `` `homonto <cmd>` `` command token from `openspec/specs/**` and fails if a token is not in the CLI's registered command set (parse `internal/cli/root.go` or `homonto --help`).
- [ ] Wire it into `scripts/gate.sh` (and thus CI + release) so a tag cannot publish on a weaker check than a PR.
- [ ] Prove it fails on the pre-change specs and passes after the delta is applied (guards the exact F5 regression).

## 3. Verify already-correct docs (no edit expected)
- [ ] Confirm `README.md:118` and `docs/guides/using-homonto.md:14` already state the `[agents]` fold with no imperative group; edit only if a stale claim remains.

## 4. Verification gate
- [ ] `openspec validate --all --no-color` passes with the deltas.
- [ ] `go build ./...` and the new CI check both green via `scripts/gate.sh` (or the narrowest equivalent).
- [ ] Delta scenarios are consistent with source (re-run task 1 checks).

## 5. Out of scope (recorded, do not implement here)
- [ ] (note only) `config.go:526` silently discarding is F35-adjacent — separate change.
- [ ] (note only) `docs/superpowers/*` historical residue is F19 — separate change.
