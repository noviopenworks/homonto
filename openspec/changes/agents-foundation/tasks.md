## 1. Config model + validation (`internal/config`)

- [ ] 1.1 (TDD RED first) Add `type Agent { Source, Version string; Targets []string; Mode string }` (toml tags) + `TargetsOrAll()` + `ModeOrDefault()`; add `Agents map[string]Agent `toml:"agents"`` to `Config`.
- [ ] 1.2 (TDD RED first) `validateAgents`: `validateKey("agents",name)`; `validSource(source)` (builtin:/local:); `mode ∈ {"",copy,link}` else error naming agent+mode; targets ∈ {claude,opencode}. Call from Parse/Load. Tests: parse full agent; defaults (version empty→unpinned, targets empty→both, mode empty→link); invalid source (https) rejected; invalid mode (symlink) rejected; unknown target rejected.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(config): [agents.<name>] lifecycle declaration model + validation`

## 2. `homonto agents list` (`internal/cli`)

- [ ] 2.1 (TDD RED first) Add `internal/cli/agents.go`: `agentsCmd()` parent (Use `agents`, shows help) + `list` subcommand that `config.Load(--config)`, sorts agent names, prints `<name>: <source>  version=<v|unpinned>  targets=<...>  mode=<...>` per agent (or `No agents declared.`). Read-only. Register `agentsCmd()` on the root.
- [ ] 2.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","list","--config",path])`: two agents → both printed sorted with source/version/targets/mode; no `[agents]` → `No agents declared.`; unpinned agent shows `unpinned`; read-only (no files written — the command only loads config).
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents list' (read-only) reports declared agents`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): a config with two `[agents.<name>]` (one pinned, one unpinned, one local mode=copy) → `homonto agents list` prints them sorted; an invalid agent source fails load.
- [ ] 3.2 Update `docs/roadmap.md` v2 status (foundation: `[agents.<name>]` model + read-only `agents list` landed; mutation/lockfile/remote deferred) + README `[agents.<name>]` example. No over-claim (no lifecycle mutation yet).
- [ ] 3.3 Commit all changes.
