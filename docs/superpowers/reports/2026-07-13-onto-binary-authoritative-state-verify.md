# Verification Report — onto-binary-authoritative-state

**Date:** 2026-07-13
**Change:** `onto-binary-authoritative-state` (ROADMAP N1, change A of A+B; Gate A onto Truth)
**Workflow:** Comet full · `verify_mode: full` (1 delta capability, 6 task groups, 19 files / +2623 in `cad5274...HEAD`)

## Result: PASS

## Full-verification checklist

| # | Check | Result |
|---|-------|--------|
| 1 | All `tasks.md` tasks `[x]` | PASS — 0 unchecked |
| 2 | Implementation matches change `design.md` | PASS — 4 units (schema / migration / CLI / classify) all delivered |
| 3 | Implementation matches Design Doc | PASS — core+typed schema, `docs/changes/<name>/onto-state.yaml`, close+archived-bool, on-read migration, both-legacy-disagree→malformed all as designed; grouping choice (flat core) recorded in Design Doc + plan |
| 4 | Capability spec scenarios pass | PASS — see scenario→test map |
| 5 | `proposal.md` goals satisfied | PASS — binary authoritative over one versioned schema; every gated mutation has a CLI command; status/doctor never drop a workspace |
| 6 | Delta spec ↔ design doc consistency | PASS — no drift; the review-driven "reject future schema_version" tightened both without contradicting |
| 7 | Design docs locatable | PASS — `docs/superpowers/specs/2026-07-13-onto-binary-authoritative-state-design.md`, plan alongside |

## Delta scenario → test

| Delta scenario (onto-binary) | Test |
|---|---|
| parse+derive from a valid versioned state | `TestMarshalParse_RoundTrip_PreservesEveryGatedField` |
| legacy state migrates on read | `TestParseAndMigrate_LegacyBinary_ToV1`, `_LegacyRich_MapsEveryGatedField` |
| disagreeing dual legacy files are malformed | `TestLoadChange_BothLegacyDisagree_IsMalformed` |
| malformed state reports a clear error | `TestValidate_MalformedEnum_Rejected`, `TestParseAndMigrate_FutureVersion_Rejected` |
| status classifies each change directory | `TestStatusCommand_ReportsValidAndInvalidChanges` |
| a deleted state file is not silently dropped | `TestStatusCommand_DeletedStateFile_IsMissingStateRow` |
| doctor: missing-state directory is a finding | `TestDoctor_MissingStateDir_IsFinding` |
| transition command sets a gated field with validation | `TestSetEnumSetters_HappyPaths`, `TestSetIsolation_BadValue_RejectedNoWrite` |
| structured read emits full state as JSON | `TestStateJSON_EmitsFullStateAndDerivedPhase` |

## Command evidence

- `go test ./internal/ontostate/... ./internal/ontocli/... -race -count=1` → 107 passed
- `go vet ./...` → no issues · `go build ./...` → success
- `openspec validate --all` → 16 passed, 0 failed

## Code review (standard, build phase)

No CRITICAL. MEDIUM forward-safety (future `schema_version` down-stamp) and LOW status-labeling **fixed** (`94f2d88`). One LOW **accepted**: a co-resident legacy `state.yaml` still folds its `Observed` over an authoritative `onto-state.yaml` — documented migration-fold behavior, `Observed` never gates (B1), so impact is limited to observational reporting; retiring the folded `state.yaml` belongs to change B (`onto-skills-shell-out`).

## Sequencing (same as N3)

The canonical `openspec/specs/onto-binary/spec.md` is corrected by the delta→main sync that **archive** performs. Archive must run before this branch lands on `main`, or `main` carries a transient stale-spec commit.

## Non-goals (unchanged, deferred)

Rewriting the `onto*` skills / deleting the markdown-only copy → change B. Semantic gate content, workflow-aware transitions, dep resolver → N2. No homonto-engine work.
