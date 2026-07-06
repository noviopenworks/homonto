# Notes: per-project-skills

Incremental checkpoint (compaction recovery). Unconfirmed items are
marked *pending*.

## Confirmed

- 2026-07-06 — **Clarification gate answered "Confirm as-is"**: goals, non-goals,
  scope boundaries, unknowns, and acceptance scenarios below are accurate.
- 2026-07-06 — **Split gate answered "Keep as one change"**: per-project skill
  scope (feature) and the Docker e2e apply smoke (tooling/validation) ship in a
  single change. Rationale recorded in proposal "Not split".
- **Scope model** (decided in plan-mode Q&A): `[skills] scope = "user" | "project"`
  in homonto.toml; default/empty = `user` (back-compat). Not per-skill, not a CLI flag.
- **Tools**: both Claude and OpenCode honor project scope.
- **Skills only relocate** — MCP servers and settings stay global.
- **Docker smoke** = e2e `apply` against a disposable `$HOME` inside a container;
  new additive CI job. Doubles as this change's end-to-end validation.
- 2026-07-06 — **Artifact-review gate answered "Approve, advance to design"**: proposal
  + tasks skeleton approved, name `per-project-skills` kept. Phase advanced open → design.

Goals / non-goals / scope / unknowns / acceptance scenarios: see proposal.md.

## Pending

- **Build-phase**: implement + test the scope-switch prune (remove inactive-scope
  location) so no orphan link remains.

## Build gate (2026-07-06)

- **Plan-ready gate answered**: isolation=`branch`, execution=`direct`, tdd=`tdd`.
  Branch `feature/20260706/per-project-skills`. One commit per task; TDD (failing test
  first) for logic-bearing tasks 1-7; docker/docs tasks 8-10 inherently direct.
- Note: `main` has pre-existing uncommitted doc edits (README.md, docs/NEXT_AGENT.md,
  docs/reviews/..., docs/roadmap.md) — NOT part of this change; commit only this change's
  files per task, never `git add -A`.

## Resolved in design (2026-07-06)

- **OpenCode project skill path = `<repo>/.opencode/skills/<name>`** (plural `skills`).
  OpenCode search order (per docs): `.opencode/skills/`, `~/.config/opencode/skills/`,
  `.claude/skills/`, `~/.claude/skills/`, `.agents/skills/`, `~/.agents/skills/`.
  Source: https://opencode.ai/docs/skills/ . **Asymmetry**: OpenCode's subpath differs by
  scope (global `.config/opencode/skills` vs project `.opencode/skills`), while Claude's
  subpath (`.claude/skills`) is identical in both scopes. => each adapter must own its
  scope→path mapping; a pure base-dir swap is WRONG for OpenCode.
- **Scope-switch orphan mechanism**: state key is `skill.<name>` (location-independent).
  On scope flip, `link.Plan(a.links())` plans a `create` at the new location, but orphan
  pruning keeps the old link because `skill.<name>` is still declared. => Apply must
  explicitly prune the inactive-scope location. `link.Remove(dst, content)` is a safe
  no-op when absent and only removes symlinks into `content/` (linker.go:43-62) — the
  right primitive.
- **Surfaces**: drift `Status()` flows through each adapter's `ObserveHashes` (claude ~193,
  opencode:201), so fixing the adapter's skill-dir makes drift scope-aware for free. Only
  `Doctor()` (status.go:104-106) independently hardcodes skill paths and needs a scope-aware
  fix; the config-location checks (status.go:87-88) are tool config dirs, unaffected.

## Grounding

- rtk 0.42.0 present; `graphify-out/` present (2026-07-04, ~2 days old — fresh
  enough). Grounding via graphify + direct file reads.
- Direct reads confirming the install-path trace:
  - `internal/cli/apply.go:21` — `home, _ := os.UserHomeDir()` (single injection point).
  - `internal/engine/engine.go` — `Build(configPath, home, contentDir)` resolves
    project root = dir(configPath) for content/ and .homonto state.
  - `internal/adapter/claude/claude.go:36-42` `links()` joins `home + .claude + skills`;
    also ObserveHashes (~193) and Apply/remove (~252).
  - `internal/adapter/opencode/opencode.go:131,201,251` join `home + .config/opencode + skills`.
  - `internal/engine/status.go:105-106` hardcodes both global skill paths.
  - `internal/config/config.go:27-29` `Skills{Own []string}`; validation loop 61-65.
  - Specs: `docs/specs/config-model.md`, `docs/specs/tool-adapters.md`,
    `docs/specs/cli-commands.md` are the surfaces to delta.
  - No Dockerfile/Makefile/scripts exist; CI (`.github/workflows/ci.yml`) smoke-tests
    only `plan`, never `apply`.

## Verify round 1 (2026-07-06) — findings → back to build

Adversarial pass (two skeptics, full mode) found:
- **FINDING 1 (real, confirmed, non-critical orphan leak — introduced here)**: switching
  scope AND removing a skill in the same apply orphans the old-scope link. The `skill.`
  delete branch removes only from `skillsDir()` (new/active scope), and the inactive-prune
  loop iterates `a.skills` (still-declared), so a de-declared skill's old-location link is
  never removed; state key deleted → status blind. **User: fix now.** Fix: delete branch
  also `link.Remove` the `inactiveSkillsDir()` location, IsManaged-guarded.
- **FINDING 2 (pre-existing gap)**: a skills-only config that loses `.homonto/state.json`
  can't rebuild state — skills have no `adopt` action, so a correct-but-unrecorded link is
  invisible to Plan and apply short-circuits. **User: fix now in this change.** Fix: add a
  skill adopt path (mirror mcp/setting/plugin) so a correct-but-unrecorded link is adopted
  into state, letting apply reconcile and rebuild.

Both fixes get a spec scenario + regression test; re-verify as round 2.

## Approaches

- **A — CONFIRMED 2026-07-06** (approach gate answered "Confirm Approach A"; switch-display
  sub-decision answered "Faithful relocate line" = A1). scope-aware adapters via a shared
  `skillpath` helper.
  New `internal/skillpath.Dir(tool, scope, home, projectRoot)` = the single source of
  truth for skill directories (claude: user `~/.claude/skills`, project `<repo>/.claude/skills`;
  opencode: user `~/.config/opencode/skills`, project `<repo>/.opencode/skills`). `config.Load`
  validates `[skills] scope` (`user|project`, empty=user). `engine.Build` resolves
  `projectRoot = dir(configPath)` and passes `scope`+`projectRoot` to both adapter
  constructors (`New(home, content, scope, projectRoot)`); Engine stores `ProjectRoot`.
  Adapters gain `skillsDir()` / `inactiveSkillsDir()` (both via the helper) and the three
  inline join sites (`links`, `ObserveHashes`, `Apply` delete) collapse to one. `Apply`
  prunes the inactive-scope location (`link.Remove`, safe no-op) so a scope flip leaves no
  orphan. `Doctor()` calls the helper for scope-aware reporting. MCP/settings paths untouched.
  Trade-off: touches config + engine + both adapters + status, but centralizes path truth
  and handles the switch correctly.
  - **Sub-decision — plan display on scope switch**: (A1, recommended) surface a faithful
    `relocate` line (Old=old location, New=new location) so the plan shows the move;
    (A2) prune silently in Apply, plan shows only the new-location `create`. Behavior is
    identical (no orphan either way); A1 is more faithful to homonto's plan→confirm ethos.
- **B (rejected) — single `skillsBase` base-dir swap** (`home` or `projectRoot`).
  Simpler, but OpenCode's project subpath (`.opencode/skills`) differs from its global
  subpath (`.config/opencode/skills`), so a pure base swap misplaces OpenCode project skills.
  Rejected on correctness.
- **C (rejected) — engine computes each adapter's full skills dir and injects it.**
  Leaks per-tool path convention into the engine and splits it from the adapter that owns
  every other path. Rejected for cohesion.
