# Verification Report: subagent-projection

**Date:** 2026-07-10
**Mode:** full (36 tasks, 3 delta capabilities, 41 changed files)
**Branch:** feature/20260710/subagent-projection (base-ref a53950f)

## Summary

| Dimension    | Status |
|--------------|--------|
| Completeness | 36/36 OpenSpec tasks ✅; 9/9 plan tasks ✅; 7 requirements all implemented |
| Correctness  | 7/7 requirements covered by code + tests; 22 subagent-specific tests, 0 failures |
| Coherence    | Follows design.md + Design Doc; one dir-scan addition documented (Implementation Notes) |

**Final assessment: All checks passed. Ready for archive.** 0 CRITICAL, 0 WARNING (the one design-doc gap was resolved by documenting it), 5 accepted SUGGESTION-level test-hardening follow-ups (SP1–SP5).

## Completeness

- OpenSpec `tasks.md`: 36/36 checked (`grep -c '- [ ]'` → 0). The single `- [ ]` match in the plan is prose in the header ("Steps use checkbox (`- [ ]`) syntax"), not a task.
- All 7 requirements of `specs/subagent-projection/spec.md` plus the MODIFIED `config-model` and `framework-expansion` deltas have implementation evidence (below).

## Correctness — requirement → implementation → test

| Requirement | Implementation | Test evidence |
|---|---|---|
| Builtin/local source resolution | `catalog.SubagentPath`, adapter `subagentSource` | `TestBuiltinSubagentLinksToSubagentCatalogRoot`, `TestLoadRejectsMissingSubagentPath` |
| Single-file **verbatim** materialization | `catalog.MaterializeSubagents` (ReadFile→WriteFile, no transform) | `TestMaterializeSubagentsWritesFileVerbatim`, `TestApplyRematerializesWhenSubagentFileMissing`, `TestMaterializeSubagentsUnknownErrors` |
| Projection into tool agent dirs (Claude `agents/`, OpenCode `agent/`) | both adapters `subagentLinks`/Plan/Apply | `TestApplyMaterializesBuiltinSubagent`, `TestBuiltinSubagentConflictNotClobbered`, `TestBuiltinSubagentPrunedWhenDeDeclared`, `TestSubagentScopeSwitchRelocatesLink` (×2 tools) |
| Adoption of pre-existing links | adapter adopt branch | `TestBuiltinSubagentAdoptsExistingLink` (×2 tools) |
| Framework `[subagents]` expansion | `catalog.ExpandSubagents`, `config.ExpandedSubagentEntriesForTool` | `TestExpandSubagentsIncludesFrameworkSubagent`, `TestExpandedSubagents{ExplicitAndTargetFilter,ExplicitVsFrameworkCollision,FrameworkInheritsScopeTargets}`, `TestLoadIndexesFrameworkSubagents` |
| Doctor verification (both tools) | `engine.doctorSubagents` | `TestDoctorReportsLinkedSubagent` |
| Bundled real subagents + minimal shared frontmatter | `catalog/subagents/{code-reviewer,codebase-explorer,comet-navigator}.md` | `TestSubagentsEmbedded`, `TestSharedFrontmatterContract` |
| (model validation, unchanged) subagent-targeted tool requires model routes | existing `EnabledModelTools`/`validateModels` | `TestLoadRequiresModelsForSubagentTargetedTool` |

**Fresh verification evidence (run 2026-07-10):**
- `go test ./... -count=1` → 0 FAIL (239 tests / 20 packages)
- `go test -race ./...` → clean (Task 9 + prior runs)
- `go vet ./...` → clean; `go build ./...` → clean; `gofmt -l .` → empty
- Dogfood: `apply --yes` links all 3 subagents into both tools; `status` → `No drift`; `doctor` → all 6 links `ok`; second apply idempotent. Independently re-verified (6 symlinks confirmed, targets into `.homonto/catalog/subagents/`).

## Coherence

- Implementation follows `design.md` and the Design Doc: mirror-command pipeline (D1), verbatim single-file materialization (D2), plural/singular dir naming via `internal/subagentpath` (D3), framework `[subagents]` expansion (D4), three real subagents with minimal shared frontmatter (D5), `subagent.*` state keys (D6).
- **One documented addition:** loose framework-agnostic builtins resolve via a global `subagents/` dir-scan in `catalog.Load()` (the D1/D5 reconciliation). Not a contradiction of the delta spec (which permits `builtin:<name>` → `catalog/subagents/<name>.md` with no framework). Recorded in the Design Doc "Implementation Notes" section during verify.
- Final whole-branch review (opus): **READY TO MERGE**, 0 Critical / 0 Important.

## Accepted follow-ups (non-blocking, SUGGESTION)

- SP1: `frontmatter_test.go` `strings.Index`→`strings.Cut` (style).
- SP2: add a dedicated unit test for the `subagents/` dir-scan (highest-value; currently covered indirectly).
- SP3: `ExpandSubagents` test checks single-framework membership (transitive/dedup proven via shared `expandResources`).
- SP4: subagent conflict test relies on Apply pre-check (weaker than command sibling; safety property still tested).
- SP5: add a test for a de-declared foreign link/real file at a subagent dst left untouched (safe by construction via `link.Remove`).

## Security

No hardcoded secrets; subagent content is verbatim markdown (no secret resolution). Path traversal rejected by `validateResourceName` (rejects `..`, `/`, non-`Base` names) before any link path is built. Symlink clobber prevented by managed-root guards in `link.*`; empty-root guard preserved in both adapters.
