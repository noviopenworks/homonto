# Proposal: per-project-skills

## Why

homonto installs owned skills into a single global location: every skill is
symlinked into `~/.claude/skills/<name>` and `~/.config/opencode/skills/<name>`,
because the install root is hardcoded to `os.UserHomeDir()` (`internal/cli/apply.go:21`,
threaded unchanged through `engine.Build` into both adapters). A developer who wants
a project's skills to live *with that project* — versioned in-repo, not leaking into
their global tool config — has no way to express it. The user wants skills configurable
**per project, not globally**.

Separately, the real `apply` is never exercised end-to-end. Go tests are fully isolated
(`t.TempDir()` + `t.Setenv("HOME", …)`), so `go test ./...` never touches the real system,
but the **compiled binary's `apply`** — which resolves the real `os.UserHomeDir()` and
mutates `~/.claude.json`, `~/.claude/settings.json`, and skill symlinks — is only ever
smoke-tested via `plan` (a dry run) in CI. Nothing proves `apply` works against a real
`$HOME`. A per-project scope is exactly the kind of behavior that deserves an end-to-end
proof against a real (but disposable) environment — so this change adds that proof too.

## What Changes

- Add `[skills] scope = "user" | "project"` to `homonto.toml`; empty/absent = `user`
  (fully back-compat, no behavior change for existing configs).
- `scope = "project"` installs owned skills under the project root (dir of `homonto.toml`)
  instead of `$HOME`: `<repo>/.claude/skills/<name>` for Claude and the OpenCode project
  skill directory for OpenCode.
- MCP servers and settings are **unaffected** — they always project into the global tool
  files. Scope governs skill symlinks only.
- `status` and `doctor` report the actual scoped install location (not a hardcoded global path).
- Switching scope re-links cleanly: the old-location symlink is pruned and the new one created.
- Add a **Docker e2e apply smoke**: a `Dockerfile` + smoke script that builds homonto and
  runs a real `apply --yes` against a throwaway `$HOME` inside a container, asserting the
  written files/symlinks (user scope) and project-scope links, and idempotency — plus a new
  additive CI job. The host system is never touched.

## Capability Impact

- **Modified**: `config-model` — `Skills` gains a `scope` field with `user | project`
  validation (delta required).
- **Modified**: `tool-adapters` — each adapter's skill link destination becomes scope-aware
  (user → `$HOME` layout, project → project-root layout); MCP/settings paths unchanged (delta required).
- **Modified**: `cli-commands` — `status`/`doctor` skill-location reporting becomes scope-aware (delta required).
- Untouched: `apply-pipeline` (plan→confirm→apply flow, atomic writes, state semantics),
  `secret-references` (resolution + hashing), `onto-workflow`.
- No new capability spec: the Docker e2e smoke is test/CI infrastructure, not a product capability.

## Not split

The clarification's split preflight proposed two changes (the scope feature and the Docker
e2e smoke). The user chose to **keep them as one change**: the Docker smoke is the natural
end-to-end validation vehicle for `apply` — including the new project-scope path this change
introduces — so building the feature and its e2e proof together keeps the proof honest and
avoids a second change whose only job is to test the first. The smoke also has standalone
value (it covers the pre-existing global `apply`, which CI never exercised).

## Grounding

- rtk 0.42.0 present; `graphify-out/` present (2026-07-04, ~2 days old — fresh). Grounding
  via graphify plus direct file reads (path trace recorded in notes.md Grounding).
- Install-root single injection point: `internal/cli/apply.go:21`; threaded via
  `internal/engine/engine.go` `Build(configPath, home, contentDir)`; skill joins in
  `internal/adapter/claude/claude.go:36-42,~193,~252` and
  `internal/adapter/opencode/opencode.go:131,201,251`; status paths
  `internal/engine/status.go:105-106`; config schema `internal/config/config.go:27-29,61-65`.
- No Dockerfile/Makefile/scripts exist; CI (`.github/workflows/ci.yml`) currently smoke-tests
  only `plan`.

## Impact

- **Files**: `internal/config/config.go`, `internal/engine/engine.go`,
  `internal/engine/status.go`, `internal/adapter/claude/claude.go`,
  `internal/adapter/opencode/opencode.go` (+ their `*_test.go`); new
  `test/docker/Dockerfile`, `test/docker/smoke.sh`, `scripts/docker-test.sh`;
  `.github/workflows/ci.yml`; spec deltas under this change's `specs/`.
- **Risks**: (1) OpenCode's project-level skill directory convention is unverified — a wrong
  path silently installs where OpenCode won't find it; design must confirm it and, if
  unconfirmable, gate OpenCode project-scope behind the verified path (Claude can ship alone).
  (2) Scope-switch must prune the old link, not orphan it — the linker only removes symlinks
  pointing into managed `content/`, which the old global link does, so this is expected to be
  safe but must be tested. (3) `.homonto/state.json` keys embed the link destination hash, so
  a scope change replans the link — intended, but assert idempotency after the switch settles.
