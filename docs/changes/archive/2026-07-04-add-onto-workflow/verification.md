# Verification Report: add-onto-workflow

- **Date:** 2026-07-04
- **Mode:** full (scale: 20 tasks, 2 delta capabilities, 59 changed files)
- **Range:** 94e3f5a (base-ref) … HEAD on `feature/20260704/add-onto-workflow`
- **Result: pass** (one recorded divergence, see below)

## Summary

| Dimension | Status |
|---|---|
| Completeness | 20/20 tasks (plan: 77 done + 5 explicitly deferred-to-archive), 12/12 requirements implemented |
| Correctness | all delta-spec scenarios evidenced (dry-runs + Go tests + live dogfood) |
| Coherence | design followed; 1 divergence recorded with user approval (Go bug fix vs "no Go changes" non-goal) |

## Checks (comet full verification)

| # | Check | Verdict | Evidence |
|---|---|---|---|
| 1 | tasks.md all checked | PASS | `grep -c '\- \[ \]' tasks.md` → 0; plan file → 0 unchecked (5 `- [>]` deferred markers for archive-phase task 16) |
| 2 | Matches high-level design.md | PASS | 8 skills, docs/ layout, state.yaml contract, presets, entry points all present as designed |
| 3 | Matches Design Doc | PASS | skill table, layout contract, state schema, gates, migration table implemented; one recorded divergence (Go fix, user-approved) |
| 4 | All spec scenarios pass | PASS | see scenario table below |
| 5 | proposal.md goals satisfied | PASS | self-contained skill set live via dogfood; migration done except archive-phase retirement |
| 6 | Delta spec vs Design Doc | PASS after resolution | drift (tool-adapters delta vs stale non-goal) resolved via Implementation Divergence section, user chose to affirm implemented state |
| 7 | Design doc locatable | PASS | docs/superpowers/specs/2026-07-04-onto-workflow-design.md |

## Scenario evidence

**onto-workflow (new capability, 11 requirements):** validated by two
fresh-context dry-run agents (full lifecycle 9/9 checks; presets + drift
8/8 checks) — results and the 11 defects they surfaced (all fixed, both
derivation-table copies kept identical) are recorded in
`validation-notes.md`. Post-fix confirmations: self-containment grep CLEAN;
gates present in every phase skill; derivation tables byte-identical
(diff → `tables-match`); all 8 symlinks resolve and the skills registered
live in the authoring session.

**tool-adapters (modified requirement, 4 scenarios):**

| Scenario | Evidence |
|---|---|
| Idempotent link creation | `TestSkillsOnlyConfigPlansAndAppliesLinks` + live `./homonto plan` → 16 creates, apply, second plan → "No changes" |
| Skills-only config still applies | same test + live apply created all 8 links (`ls -l ~/.claude/skills/onto*`) |
| Relative content dir → absolute targets | `TestRelativeContentDirResolvesAgainstConfig` (engine) |
| Conflict reported, not clobbered | `internal/link/linker_test.go` + `link.Plan` conflict error path |

## Regression

`go test ./...` → 48 passed in 14 packages, 0 failures (fresh at verify).
`go build` → exit 0. `./homonto status` → "No drift."
`./homonto doctor` → all 8 skills ok (pass-not-found warning pre-exists,
unrelated).

## Security

No hardcoded secrets introduced (change is markdown + link-planning logic;
secret paths untouched — `internal/secret` unchanged per git diff).

## Deviations

1. **Go source changes vs original non-goal** — approved scope amendment
   (user, 2026-07-04); recorded in proposal.md Modified Capabilities and
   Design Doc Implementation Divergence. Impact: `internal/link`,
   both adapters, `internal/engine`; covered by 3 new tests.
