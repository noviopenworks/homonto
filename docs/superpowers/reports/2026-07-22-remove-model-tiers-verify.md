# Verification Report: remove-model-tiers

- **Date:** 2026-07-22
- **Mode:** full (27 tasks, 47 changed files, 1 delta capability)
- **Branch:** `feature/20260721/remove-model-tiers`
- **Base ref:** `eb3fe5c`

## Summary

| Dimension    | Status                                                     |
|--------------|------------------------------------------------------------|
| Completeness | 27/27 tasks checked; 4/4 delta requirements implemented     |
| Correctness  | 9/9 delta scenarios covered by code + tests                 |
| Coherence    | D1–D4 followed; D5 diverged, recorded in the Design Doc     |

**Assessment:** No CRITICAL issues. One WARNING resolved by recording the
divergence (operator choice). Ready for archive.

## Evidence (fresh runs, 2026-07-22)

| Check                | Command                | Result                          |
|----------------------|------------------------|---------------------------------|
| Build                | `go build ./...`       | Success                         |
| Vet                  | `go vet ./...`         | No issues                       |
| Tests                | `go test ./...`        | 961 passed, 42 packages         |
| Config health        | `homonto doctor`       | exit 0 (warnings only)          |
| Residual tier symbols| grep `TierNames`/`Roles`/`Tiers`/`routeModelSetting` in non-test Go | none |
| Residual `role:`     | grep `catalog/subagents/*.md` | none                     |

`homonto doctor` warnings are pre-existing and unrelated to this change: stale
`.homonto/` catalog version, `pass` not on PATH, and `unknown framework
"comet"` (the catalog no longer ships comet — ADR 0015; dogfooding deferred).

## Completeness

- `tasks.md`: 27/27 `[x]`. Tasks 5.2 and 5.3 carry explicit deviation notes.
- Delta spec `specs/agent-models/spec.md`: all 4 requirements implemented.

## Correctness — scenario coverage

| Scenario | Evidence |
|----------|----------|
| All agents modeled — load succeeds | `homonto doctor` exit 0; `TestLoadAcceptsModelWithoutEffortOrVariant` |
| Missing agent model — load fails naming offender | `TestLoadRequiresPerToolModelForEnabledSubagents`, `TestLoadRequiresModelsForSubagentTargetedTool` |
| Framework declares existence, config declares model | `TestLoadRequiresModelsForFrameworkExpandedSubagents`, `TestLoadRequiresModelForFrameworkAgentOutsideExplicitAliasTargets` |
| Legacy tier block rejected | `TestLegacyModelTierRejected`, `TestUnknownModelTierRejected`; `config/validate.go` `rejectLegacyModels` |
| `role:` frontmatter derives no model | `agentfm.Homonto` has no `Role` field — YAML drops it as unknown (`agentfm.go:50-61`) |
| No main model written when settings absent | `TestDesired_NoMainModelWhenSettingsAbsent` (claude), `TestDesiredSettings_NoMainModelWhenSettingsAbsent` (opencode) |
| Explicit main-model override still projected | `adapter/opencode/models_test.go` settings-path assertions |
| Invalid Claude effort rejected | `TestLoadValidatesModelSpecPerTool` |
| Effort on OpenCode rejected | `TestLoadValidatesModelSpecPerTool` |

`Config.Models` survives solely as the legacy-block detector feeding
`rejectLegacyModels` — required by the "legacy tier block rejected"
requirement, not a surviving tier reader.

## Coherence — design adherence

- **D1 (remove tier system):** followed — no `Role`/`TierNames`/`Tiers`/`Roles`.
- **D2 (models in `[subagents.<name>.<tool>]`):** followed.
- **D3 (stop managing the main model):** followed — `routeModelSetting` and the
  route-derived `routeSettings` halves are gone; explicit `[settings.*]` intact.
- **D4 (must-declare validation):** followed — errors name subagent and tool.
- **D5 (drop redundant source blocks):** **DIVERGED.** This repo declares
  `frameworks.comet`, which the catalog no longer ships, so no framework
  declares `onto-reviewer` / `onto-explorer`; dropping the source blocks would
  delete the agents. 4 model blocks ship instead of the projected 18. Operator
  chose to record the divergence: an "Implementation Divergence" section was
  appended to `docs/superpowers/specs/2026-07-21-remove-model-tiers-design.md`
  (`9cf6b88`). No delta-spec requirement depends on D5.

## Post-build changes reviewed

Two commits landed after the build-phase review:

- `7295029` — render context becomes a pointer (nil = catalog projection, no
  model line; non-nil = production render that must resolve a model), plus a
  `Targets` set so unselected tool variants are skipped and stale variant files
  removed. Targets participate in the render fingerprint, so selection changes
  re-plan. Covered by `TestRenderMissingModelOverrideErrorsWithRenderContext`
  and `TestRenderNilContextRemainsLenientForCatalogProjection`.
- `4a58bb0` / `9cf6b88` — artifacts only (ADR draft, task closure, divergence).

No correctness, security, or edge-case issue found: the change tightens an
error path rather than loosening one, and no secrets or unsafe operations were
introduced.

## Follow-up (out of scope)

`homonto.toml` still declares `frameworks.comet`, removed from the catalog by
ADR 0015. This produces six `homonto doctor` warnings and is the reason D5's
premise failed. Tracked as repo hygiene, already noted under Risks in the
design; dogfooding is deferred to v1 by prior decision.
