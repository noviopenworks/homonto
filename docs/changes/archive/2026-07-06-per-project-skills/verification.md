# Verification Report: per-project-skills

- **Date:** 2026-07-06
- **Mode:** full (why: `workflow: full`, new capability, diff touches >5 files in base..HEAD)
- **Range:** e14db90..56b27ac on `feature/20260706/per-project-skills`
- **Result: pass**

Two verify rounds. Round 1 found one real defect (FINDING 1) and one pre-existing gap
(FINDING 2); the user chose to fix both (see notes.md), which sent the change back to build.
Round 2 re-verified with two fresh adversarial skeptics that could not refute the fixes.

## Scenario evidence

All commands run fresh in this round (2026-07-06). Full suite: `go test ./...` → **144 passed
in 16 packages**; `go test -race ./...` → **144 passed**; `gofmt -l .` empty; `go vet ./...`
clean; `go mod tidy -diff` clean.

| Requirement / Scenario | Verdict | Evidence |
|---|---|---|
| config-model: Absent scope defaults to user | pass | `go test ./internal/config -run TestLoadSkillScope` → PASS (asserts absent/empty → `"user"`) |
| config-model: Project scope parsed | pass | same test → PASS (`scope="project"` → `"project"`) |
| config-model: Invalid scope is rejected | pass | same test → PASS (`scope="global"` errors, names `"global"` + `user`/`project`) |
| skillpath: per-tool per-scope mapping | pass | `go test ./internal/skillpath` → PASS (all 4 tool×scope dirs; non-`project`→user) |
| tool-adapters: Claude project scope links under project root | pass | `go test ./internal/adapter/claude -run TestProjectScopeLinksUnderProjectRoot` → PASS (`<proj>/.claude/skills`, none under home) |
| tool-adapters: OpenCode project scope links under project root | pass | `go test ./internal/adapter/opencode -run TestProjectScopeLinksUnderProjectRoot` → PASS (`<proj>/.opencode/skills`) |
| tool-adapters: MCP/settings unaffected by scope | pass | conformance skeptic (round 1): at project scope, MCP still only in `~/.claude.json` and `~/.config/opencode/opencode.jsonc`; adapter config paths use `a.home` unconditionally |
| tool-adapters: Switching scope relocates the link | pass | `go test ./internal/adapter/... -run TestScopeSwitchRelocatesLink` → PASS; docker smoke shows `~ skill.demo: <home> -> <proj> -> <src>` |
| tool-adapters: Relocation prune only touches homonto's own link | pass | `go test ./internal/adapter/opencode -run TestRelocationPruneLeavesForeignFile` → PASS (foreign file preserved, apply exit 0) |
| tool-adapters: De-declaring a skill while switching scope leaves no orphan (FIX 1) | pass | `go test ./internal/adapter/{claude,opencode} -run TestRemoveAndSwitchLeavesNoOrphan` → PASS (both adapters, both directions per round-2 skeptic) |
| tool-adapters: Correct-but-unrecorded skill link is adopted (FIX 2) | pass | `go test ./internal/adapter/{claude,opencode} -run TestSkillAdoptRebuildsState` → PASS; `go test ./internal/engine -run TestSkillsOnlyRebuildsLostState` → PASS (state.json deleted → rebuilt → prunable) |
| cli-commands: doctor project scope checked at project location | pass | `go test ./internal/engine -run TestDoctorProjectScopeChecksProjectLocation` → PASS; docker smoke doctor reports project links `ok` |
| cli-commands: doctor missing skill / missing OpenCode link | pass | existing `TestDoctorFlagsMissingSkillContent`, `TestDoctorChecksOpenCodeSkillLink` → PASS (unchanged behavior) |
| End-to-end (real binary): user+project apply, idempotency, switch, doctor | pass | `scripts/docker-test.sh` → **SMOKE PASS** (disposable `$HOME`, host untouched) |

## Design conformance

Walked `design.md`'s key decisions against the implementation:

1. **Skill scope is config, skills-only.** `config.Skills.Scope` added + validated; `skillpath`
   maps only skill dirs. Adapter MCP/settings paths (`claudeJSON`, `settingsJSON`, `cfgFile`)
   use `a.home` unconditionally — confirmed by both round-1 skeptics. Conforms.
2. **Shared `skillpath` helper owns the path convention.** Single `skillpath.Dir`; both adapters
   and `Doctor` call it; no path string duplicated. Conforms.
3. **Scope switch is an explicit relocate.** Plan renders the relocate; Apply prunes the
   inactive link (IsManaged-guarded). Conforms; the two round-1 findings hardened it (delete
   branch now also prunes the inactive location; adopt rebuilds lost state) — both recorded in
   design.md "Error handling".
4. **`WithScope` builder** (build-time refinement of the 4-arg `New`) keeps the existing
   `New(home, content)` signature so ~40 existing user-scope tests stayed as regression
   coverage. Recorded in design.md.

No unexplained deviations.

## Adversarial pass

Full mode: two skeptics per round, dispatched in parallel, prompted to refute.

**Round 1** (metrics.verify_rounds bumped to 1):
- Conformance skeptic — could not refute the six core claims (real-binary tests): no-regression
  for empty scope, project-root links for both tools with OpenCode's distinct `.opencode/skills`,
  MCP/settings scope-independence, no-orphan + foreign-file-safety on switch, scoped doctor,
  faithful relocate line. **No CRITICAL.** Notes: empty-plan short-circuit (unreachable via
  homonto's own ops) and cosmetic triple-arrow render.
- Robustness skeptic — **FINDING 1** (real, non-critical orphan on remove+switch in one apply)
  and **FINDING 2** (pre-existing: skills-only lost `state.json` can't rebuild). Triaged: both
  fixed per the failure gate (user's choice). Other attacks held (degenerate projectRoot==home,
  toggling, drift/recovery, conflicts, symlinked paths, fresh-$HOME skills-only, order).

**Round 2** (metrics.verify_rounds bumped to 2) — two focused skeptics on the fixes:
- FIX 1 skeptic — **could not break it**: orphan fully closed both directions/both adapters;
  plain-removal regression intact (active-path conflict still reported, inactive foreign file
  untouched); error guard skips inactive prune on active conflict; no false drift; degenerate
  projectRoot==home handled. One impractical-to-trigger self-healing error surface (speculation).
- FIX 2 skeptic — **could not break it**: adopt hash byte-identical to a fresh link (true noop,
  no re-adopt loop); adopt fires only for a correct-but-unrecorded link (not for recorded/wrong/
  missing/foreign); scope-correct; coexists with relocate without double-emission; the CLI
  reconcile path runs ("Reconciled 2 pre-existing resource(s)"). One cosmetic note (inactive
  prune not surfaced in plan on adopt-only runs).

## Regression

`go test ./...` → **144 passed in 16 packages** (fresh, this round).
`go test -race ./...` → **144 passed**.
`gofmt -l .` → empty. `go vet ./...` → clean. `go mod tidy -diff` → clean.
`scripts/docker-test.sh` → **SMOKE PASS** (real `apply` in a disposable container).

## Deviations

None. (Three non-defect observations were raised and dismissed: the cosmetic triple-arrow
relocate render; the inactive-scope prune not being surfaced in the plan on an adopt-only or
no-op run — desirable, IsManaged-guarded cleanup, only touches homonto's own symlinks; and an
impractical-to-trigger, self-healing error surface if an inactive `link.Remove` fails after the
active one succeeds. None require a fix; none change behavior a scenario asserts.)
