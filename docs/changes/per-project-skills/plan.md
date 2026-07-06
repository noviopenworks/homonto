# Plan: per-project-skills

Design: `design.md` (Status: Confirmed 2026-07-06). One commit per task.

## Task 1 — Add `[skills] scope` to the config model with validation

- [x] done
- Files: `internal/config/config.go`, `internal/config/config_test.go`
- Do: add `Scope string \`toml:"scope"\`` to the `Skills` struct (config.go:27-29). In
  `Load`, after parsing, normalize empty → `"user"` and reject any value other than
  `user`/`project` with a named error (sibling to the existing target/reserved-key checks).
  Keep the `skills.own` bare-name loop. Per design "Architecture" (config bullet).
- Verify: `go test ./internal/config/` — new cases: `scope="project"` parsed, absent →
  `user`, `scope="global"` rejected naming the value + valid set.

## Task 2 — Add the `skillpath` helper (single source of path truth)

- [x] done
- Files: `internal/skillpath/skillpath.go`, `internal/skillpath/skillpath_test.go`
- Do: `func Dir(tool, scope, home, projectRoot string) string` mapping per design:
  claude→`.claude/skills` (root = home for user, projectRoot for project);
  opencode→user `.config/opencode/skills`, project `.opencode/skills`. Non-`project`
  scope treated as user. Add `func otherScope(scope string) string` (or inline).
- Verify: `go test ./internal/skillpath/` — all four (tool × scope) mappings + non-`project`
  normalized to user.

## Task 3 — Thread scope + projectRoot through the engine

- [x] done
- Files: `internal/engine/engine.go` (+ adjust any direct `engine.Build`/adapter `New`
  call sites and their tests as needed to compile)
- Do: in `Build`, compute `projectRoot, _ := filepath.Abs(filepath.Dir(configPath))`, read
  `scope := cfg.Skills.Scope`, pass both into `claude.New(...)`/`opencode.New(...)` (signatures
  updated in Task 4), add `ProjectRoot` field to `Engine` and set it. MCP/settings untouched.
- Verify: `go build ./...` (compiles after Task 4 signatures land — sequence 3+4 together if
  needed); `go test ./internal/engine/` green.

## Task 4 — Make both adapters scope-aware (`skillsDir`, collapse join sites)  (risk: high)

- [x] done
- Files: `internal/adapter/claude/claude.go`, `internal/adapter/opencode/opencode.go`, and
  their `*_test.go` (constructor calls + new project-scope cases)
- Do: add `scope`/`projectRoot` fields; `New(home, content, scope, projectRoot string)`;
  `skillsDir()` = `skillpath.Dir(<tool>, a.scope, a.home, a.projectRoot)` and
  `inactiveSkillsDir()` = same with `otherScope`. Replace the three inline joins (`links()`,
  the `skill.` branch of `ObserveHashes`, the `skill.` delete branch of `Apply`) with
  `filepath.Join(a.skillsDir(), name)`. Per design "Architecture" (adapter bullet).
- Verify: `go test ./internal/adapter/...` — user-scope regression unchanged; new: with
  `scope="project"`, `links()`/plan/apply land the symlink under `<projectRoot>/.claude/skills`
  and `<projectRoot>/.opencode/skills`.

## Task 5 — Scope-switch relocate (plan) + prune (apply), both adapters  (risk: high)

- [x] done
- Files: `internal/adapter/claude/claude.go`, `internal/adapter/opencode/opencode.go`, tests
- Do: in `Plan`, when a skill's active-location link is a create AND
  `inactiveSkillsDir()/<name>` holds a managed symlink into `content/`, emit the change as a
  relocate (`Action: "update"`, `Old`=inactive dst, `New`=active dst). In `Apply`, before
  creating active links, `link.Remove(filepath.Join(a.inactiveSkillsDir(), name), a.content)`
  for each owned skill (no-op when absent; conflict-safe). Per design "Architecture" + the
  tool-adapters delta "Skill scope relocation leaves no orphan".
- Verify: `go test ./internal/adapter/...` — flip scope: plan shows a relocate; apply removes
  old link + creates new; old path gone, new present; second apply noop. Conflict-safety: a
  real file at the inactive path is left untouched and reported.

## Task 6 — Scope-aware `doctor`

- [x] done
- Files: `internal/engine/status.go`, `internal/engine/status_test.go`
- Do: replace the hardcoded skill paths in `Doctor` (lines 104-106) with
  `skillpath.Dir("claude"/"opencode", e.Cfg.Skills.Scope, e.Home, e.ProjectRoot)`. Leave the
  config-location checks (87-88) and drift `Status()` untouched.
- Verify: `go test ./internal/engine/` — project-scope apply then `doctor` reports the skill
  links `ok` at the project location.

## Task 7 — Engine e2e: project scope idempotency + clean switch

- [ ] done
- Files: `internal/engine/e2e_test.go` (or a new `scope_e2e_test.go`)
- Do: apply with `scope="project"` over a temp home + temp repo; assert links under the repo
  dir, `status` clean, second apply idempotent; then switch `project→user`, re-apply, assert
  the project link is gone and the home link present (no orphan).
- Verify: `go test ./internal/engine/` green.

## Task 8 — Docker e2e apply smoke + wrapper

- [ ] done
- Files: `test/docker/Dockerfile`, `test/docker/smoke.sh`, `scripts/docker-test.sh`
- Do: `Dockerfile` on `golang:1.23` builds homonto to `/usr/local/bin/homonto` and runs
  `smoke.sh`. `smoke.sh` (with a throwaway `$HOME`): write a `homonto.toml` owning an existing
  `content/skills/*` skill (e.g. `onto`), `apply --yes`, assert `~/.claude/skills/<name>`
  symlink + `~/.claude.json` + `~/.claude/settings.json`; `status`/`doctor` clean; second
  `apply` prints "No changes"; then `scope="project"`, apply, assert links under the repo copy
  and NOT under `$HOME`. `scripts/docker-test.sh` = `docker build` + `docker run --rm`, non-zero
  on failure. Per design "Testing strategy" (docker) + tasks 5.1-5.3.
- Verify: `scripts/docker-test.sh` exits 0.

## Task 9 — Additive `docker-e2e` CI job

- [ ] done
- Files: `.github/workflows/ci.yml`
- Do: add a `docker-e2e` job (ubuntu-latest) running `scripts/docker-test.sh`; leave the
  existing `test` job unchanged.
- Verify: `yaml` parses; job runs `scripts/docker-test.sh` (validated via the local docker run).

## Task 10 — Docs: README + guide for `scope`

- [ ] done
- Files: `README.md`, `docs/guides/` (the relevant using-homonto guide)
- Do: document `[skills] scope = "user" | "project"`, the per-tool project locations, and the
  scope-switch relocate behavior.
- Verify: manual read; `gofmt`/`go vet` unaffected.

## Task 11 — Validation (the change proving itself)

- [ ] done
- Files: — (runs the full suite + docker smoke)
- Do: fresh `gofmt -l .` (clean), `go vet ./...`, `go mod tidy -diff`, `go build ./...`,
  `go test ./...`, `go test -race ./...`, then `scripts/docker-test.sh`. Confirm the host
  `~/.claude/` is untouched by the container run.
- Verify: all commands pass with output captured for `verification.md`.
