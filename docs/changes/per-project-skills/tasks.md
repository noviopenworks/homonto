# Tasks: per-project-skills

## 1. Config model

- [x] 1.1 Add `scope` to the `Skills` struct and validate it in `config.Load`
      (`user | project`, empty = `user`, else named error); keep the bare-name
      check on `skills.own`.

## 2. Scope threading

- [x] 2.1 Add `internal/skillpath.Dir(tool, scope, home, projectRoot)` — the single source
      of truth (claude: `.claude/skills`; opencode: user `.config/opencode/skills`, project
      `.opencode/skills`), with tests for all four (tool × scope) mappings.
- [x] 2.2 `engine.Build`: resolve `projectRoot = dir(configPath)`, read `cfg.Skills.Scope`,
      pass `scope`+`projectRoot` into both adapters (`.WithScope(...)`), store `Engine.ProjectRoot`;
      MCP/settings paths untouched.

## 3. Adapters

- [x] 3.1 Claude adapter: add `scope`/`projectRoot` fields + `WithScope` + `skillsDir()`
      (via `skillpath`); collapse the three join sites (`links`, `ObserveHashes`, Apply/remove).
- [x] 3.2 OpenCode adapter: same, using `.opencode/skills` for project scope.
- [x] 3.3 Relocate + prune (both adapters): Plan renders a scope switch as a relocate
      (`skill.<name>` old→new); Apply prunes the inactive-scope link (`link.Remove`, no-op when
      absent, conflict-safe) before creating the active link — no orphan.

## 4. Status / doctor

- [x] 4.1 `Doctor` (`internal/engine/status.go`): use `skillpath.Dir` for both tool skill
      paths so a project-scoped install isn't reported missing. (Drift `Status()` needs no
      change — it flows through the now-scope-aware `ObserveHashes`.)

## 5. Docker e2e apply smoke

- [x] 5.1 Add `test/docker/Dockerfile` (golang:1.23) that builds homonto and runs the smoke.
- [x] 5.2 Add `test/docker/smoke.sh`: apply against a disposable `$HOME`, assert user-scope
      files/symlinks + idempotency, then project-scope links land in the repo (not `$HOME`).
- [x] 5.3 Add `scripts/docker-test.sh` wrapper (build image + run, non-zero on failure).
- [x] 5.4 Add an additive `docker-e2e` job to `.github/workflows/ci.yml`.

## 6. Specs & guides

- [ ] 6.1 Write delta specs under this change's `specs/` for `config-model`,
      `tool-adapters`, `cli-commands` (MODIFIED requirements + scenarios).
- [ ] 6.2 Update `docs/guides/` / `README.md` for the new `scope` setting.

## 7. Validation

- [ ] 7.1 Go tests green: config scope parse/reject/default; adapter project-scope link
      locations (Claude + OpenCode) with user-scope regression; engine e2e apply with
      `scope="project"` idempotent + clean scope-switch (remove old, create new); `status`/
      `doctor` clean. Plus `gofmt`/`vet`/`go mod tidy -diff`/`-race`.
- [ ] 7.2 `scripts/docker-test.sh` exits 0; manual confirmation that the host `~/.claude/`
      is untouched and project-scope links land in the repo copy inside the container.
