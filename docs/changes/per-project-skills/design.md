# Design: per-project-skills

Status: Confirmed
Confirmed: 2026-07-06 (Approach A — scope-aware adapters via a shared `skillpath` helper; faithful relocate line on scope switch)

## Summary

Add `[skills] scope = "user" | "project"` to the config model (empty/absent = `user`,
fully back-compat). A new `internal/skillpath` package is the single source of truth for a
skill's install directory as a function of `(tool, scope, home, projectRoot)`. `engine.Build`
resolves `projectRoot = dir(homonto.toml)` and threads `scope`+`projectRoot` into both adapter
constructors; each adapter's three inline skill-path joins collapse to one `skillsDir()` call.
On a scope switch the plan shows a faithful `relocate` line and `apply` prunes the
inactive-scope link (`link.Remove`, a safe no-op), so no orphan is left. `doctor` reports the
scope-resolved location. MCP servers and settings are untouched — scope governs skill symlinks
only. Rejected: **B** a single base-dir swap (OpenCode's project subpath `.opencode/skills`
differs from its global `.config/opencode/skills`, so a base swap misplaces OpenCode skills);
**C** engine-side path computation (leaks per-tool path convention out of the adapters).

## Goals / Non-Goals

**Goals:** a project can keep its owned skills under the project root instead of `$HOME`,
for both Claude (`<repo>/.claude/skills/`) and OpenCode (`<repo>/.opencode/skills/`); default
behavior is unchanged; scope switches re-link cleanly with no orphan; status/doctor/drift all
reflect the actual location. **Non-goals:** relocating MCP servers or settings (always global);
per-skill scope; a CLI-flag override; changing the plan→confirm→apply, secret, or state model.

## Architecture

New + changed pieces (no MCP/settings path touched anywhere):

- **`internal/config/config.go`** — `Skills` gains `Scope string \`toml:"scope"\``.
  `Load` normalizes empty → `"user"` and rejects anything other than `user`/`project` with a
  named error (sibling to the existing target/settings validation). The bare-name check on
  `skills.own` stays (names still become path components, now possibly under a project root).

- **`internal/skillpath/skillpath.go`** (new) — the one place the path convention lives:
  ```
  Dir(tool, scope, home, projectRoot) string
    claude   + user    -> <home>/.claude/skills
    claude   + project -> <projectRoot>/.claude/skills
    opencode + user    -> <home>/.config/opencode/skills
    opencode + project -> <projectRoot>/.opencode/skills
  ```
  A non-`project` scope is treated as `user` (defense in depth; `config.Load` already
  normalized). Both adapters and `engine.Doctor` call it — no path string is duplicated.

- **`internal/engine/engine.go`** — `Build` computes `projectRoot, _ := filepath.Abs(filepath.Dir(configPath))`
  (the same directory already used to anchor `content/` and `.homonto/`), reads
  `scope := cfg.Skills.Scope`, constructs `claude.New(home, contentDir).WithScope(scope,
  projectRoot)` and the opencode equivalent, and stores `ProjectRoot` on `Engine` (used by
  `Doctor`). (Refinement during build: a `WithScope` builder rather than a 4-arg `New` keeps the
  established `New(home, content)` signature intact, so the ~40 existing adapter-test call sites
  remain untouched user-scope regression coverage; empty scope = user.)

- **`internal/adapter/claude/claude.go` & `internal/adapter/opencode/opencode.go`** —
  each adapter gains `scope` and `projectRoot` fields (set via `WithScope`) and two helpers:
  `skillsDir()  = skillpath.Dir(<tool>, a.scope, a.home, a.projectRoot)` and
  `inactiveSkillsDir() = skillpath.Dir(<tool>, skillpath.Other(a.scope), a.home, a.projectRoot)`.
  The three current join sites — `links()`, the `skill.` branch of `ObserveHashes`, and the
  `skill.` delete branch of `Apply` — all switch to `filepath.Join(a.skillsDir(), name)`.
  - **Plan (relocate)**: after `link.Plan(a.links())`, for each owned skill whose new-location
    link is a `create` AND whose `inactiveSkillsDir()/<name>` currently holds a managed symlink
    into `content/`, render the change as a relocate — `Action: "update"`, `Old:
    <inactive-dst>`, `New: <active-dst>` — instead of a bare create, so the move is visible.
  - **Apply (prune)**: before creating the active links, for each owned skill call
    `link.Remove(filepath.Join(a.inactiveSkillsDir(), name), a.content)` (no-op when absent;
    conflict-safe — only removes symlinks into `content/`). Then create active links as today.
    State key stays `skill.<name>` (location-independent); its Applied hash is recomputed from
    the new destination, so the next plan/apply is a steady-state noop.

- **`internal/engine/status.go`** — `Doctor` replaces the two hardcoded skill paths
  (lines 104-106) with `skillpath.Dir("claude"/"opencode", e.Cfg.Skills.Scope, e.Home,
  e.ProjectRoot)`. The config-location checks (87-88, tool config dirs) are unchanged. Drift
  `Status()` needs no change — it flows through each adapter's now-scope-aware `ObserveHashes`.

## Key decisions

1. **Skill scope is config, skills-only** — `[skills] scope`, default `user`; MCP/settings
   stay global. (ADR: `adopt-skill-install-scope`.)
2. **A shared `skillpath` helper owns the path convention** — because OpenCode's project vs
   global subpaths differ, the mapping can't be a base swap and must not be duplicated across
   the three adapter sites + doctor. (Same ADR — architectural rationale.)
3. **Scope switch is an explicit relocate** — plan shows it, apply prunes the inactive
   location; no orphan. (Same ADR — behavior rationale.)

## Error handling

- Invalid `scope` value → `config.Load` returns a named error; `plan`/`apply`/`status` all
  fail fast before any write (load happens first in `engine.Build`).
- A real file or foreign symlink at either the active or inactive skill path → `link.Link` /
  `link.Remove` already report a conflict and never clobber; scope work inherits that safety.
- Missing inactive-location link on a scope switch (e.g. first apply, or manual cleanup) →
  `link.Remove` returns nil (absent is fine); relocate degrades to a plain create.
- **De-declare + scope switch in one apply** (verify round 1, FINDING 1): the `skill.`
  delete branch prunes both the active AND the inactive-scope location (IsManaged-guarded),
  because a removed skill's link may physically sit at the now-inactive scope — otherwise it
  would orphan. A foreign file at either location is left untouched.
- **Lost `state.json` for a skills-only config** (verify round 1, FINDING 2): a correct link
  that link.Plan omits (nothing to change) but that state doesn't record is emitted as an
  `adopt`, so apply runs and rebuilds state (mirroring mcp/setting/plugin adoption). Without
  this a skills-only repo could never reconstruct state or prune a later orphan.

## Testing strategy

- **config** (`config_test.go`): `scope="project"`/`"user"`/absent parse; invalid value
  rejected with the offending value named.
- **skillpath** (`skillpath_test.go`): all four (tool × scope) mappings; non-`project`
  normalized to user.
- **adapters** (`claude_test.go`, `opencode_test.go`): with `scope="project"`, `links()` /
  plan / apply land the symlink under `<projectRoot>/.claude/skills` and
  `<projectRoot>/.opencode/skills`; user-scope regression unchanged; a scope flip plans a
  relocate and apply removes the old link + creates the new one (assert old path gone, new
  present, second apply noop).
- **engine e2e** (`e2e_test.go`): apply with `scope="project"` is idempotent; `status` clean;
  `doctor` reports the project location `ok`; a `project→user` switch leaves no orphan.
- **docker** (`test/docker/smoke.sh`): real `apply --yes` against a disposable `$HOME` proves
  user-scope files/links + idempotency, then project-scope links land in the repo copy and not
  in `$HOME`; host untouched.
- Repo hygiene: `gofmt`, `go vet`, `go mod tidy -diff`, `go test ./...`, `go test -race ./...`.

## Grounding

- OpenCode skill search order (project `.opencode/skills/`, global `~/.config/opencode/skills/`,
  plus `.claude`/`.agents` compat): https://opencode.ai/docs/skills/ .
- Path trace: `internal/cli/apply.go:21`, `internal/engine/engine.go:31-59`,
  `internal/adapter/claude/claude.go:36-42,~193,~252`,
  `internal/adapter/opencode/opencode.go:128-134,199-207,248-252`,
  `internal/engine/status.go:96-114`, `internal/config/config.go:27-29,61-65`.
- `link.Remove` no-op-on-absent + content-root safety: `internal/link/linker.go:43-62`.
- State key is `skill.<name>` (location-independent): `internal/adapter/opencode/opencode.go:299`,
  claude equivalent — the reason a scope switch orphans without an explicit prune.
