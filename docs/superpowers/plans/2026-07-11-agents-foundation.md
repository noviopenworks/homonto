---
change: agents-foundation
design-doc: docs/superpowers/specs/2026-07-11-agents-foundation-design.md
base-ref: b4406fdf853d2ed0ca8996a192cb29262314098e
---

# Plan: agents foundation (v2 #1)

`[agents.<name>]` lifecycle model + read-only `homonto agents list`. Read-only,
no mutation/lockfile/projection. See the Design Doc for exact code. TDD.

## Task 1: config model + validation (`internal/config`)

- [ ] 1.1 (TDD RED first) `type Agent { Source, Version string; Targets []string; Mode string }` (toml tags) + `TargetsOrAll()` + `ModeOrDefault()`; `Config.Agents map[string]Agent`.
- [ ] 1.2 (TDD RED first) `validateAgents` (validateKey + validSource builtin:/local: + mode ∈ {"",copy,link} + targets ∈ {claude,opencode}), called from Parse/Load. Tests: full agent parses; defaults (unpinned/both/link); invalid source (https) rejected; invalid mode (symlink) rejected; unknown target rejected.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [agents.<name>] lifecycle declaration model + validation`

## Task 2: `homonto agents list` (`internal/cli`)

- [ ] 2.1 (TDD RED first) `internal/cli/agents.go`: `agentsCmd()` parent + read-only `list` (config.Load, sort names, print `<name>: <source>  version=<v|unpinned>  targets=<...>  mode=<...>` or `No agents declared.`). Register on root.
- [ ] 2.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","list","--config",p])`: two agents sorted w/ fields; unpinned shows `unpinned`; no `[agents]` → `No agents declared.`; read-only. Mirror status_test.go harness.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents list' (read-only) reports declared agents`

## Task 3: Regression and docs

- [ ] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): two `[agents.<name>]` (pinned builtin + unpinned local mode=copy) → `homonto agents list` prints sorted; invalid source fails load.
- [ ] 3.2 Update `docs/roadmap.md` v2 status + README `[agents.<name>]` example. No over-claim (no mutation yet).
- [ ] 3.3 Commit all changes.
