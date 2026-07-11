# Verification Report: onto-phase-gates

**Date:** 2026-07-10 · **Mode:** full · **Branch:** feature/20260710/onto-phase-gates (base-ref 6a41f8a)

## Summary

| Dimension | Status |
|---|---|
| Completeness | 13/13 OpenSpec tasks ✅; 13/13 plan tasks ✅; all 3 delta requirements implemented |
| Correctness | 3/3 requirements + 8 scenarios covered by code + tests; ontostate+ontocli suites, 0 failures |
| Coherence | Follows design.md + Design Doc; isolated from homonto; onto binary #3b |

**Final: All checks passed. Ready for archive.** 0 CRITICAL, 0 WARNING. One Important review finding (inverted gate semantics — gate checked the NEXT phase's artifacts instead of the CURRENT phase's) was FIXED during build: delta spec, design doc, code, and tests all corrected to current-phase-deliverables-gate-leaving semantics, verified end-to-end. Minor follow-ups OF-g1/OF-g2 + cosmetics accepted.

## Correctness — requirement → implementation → test (fresh run 2026-07-10)

| Requirement | Implementation | Evidence |
|---|---|---|
| Per-phase required artifacts (cumulative) | `ontostate.RequiredArtifacts` | per-phase set tests; ValidateSkeleton tightens (verify needs verification.md) |
| onto advance gates phase transitions (current-phase deliverables) | `ontocli.advance.go` + `ontostate.NextPhase`/`TasksAllChecked` | open→design (no design.md); design→build refused w/o design.md; build→verify blocked by unchecked task; past-close error; success writes phase (Load-back); no-write-on-failure |
| dirty worktree blocks close | `ontocli.worktreeDirty` (git status --porcelain) | dirty WARNs on normal advance; BLOCKs verify→close (phase unchanged) |

**Fresh gates:** `go build ./...` clean (both binaries); `go test ./... -count=1` → 0 FAIL (304 tests); `go test -race` on the two packages clean; `go vet ./...` clean; `gofmt -l .` empty; `go mod tidy` clean (no new deps). **E2E (temp git workspace):** `onto new demo` → `onto advance` walks open→design→build→verify through the gates; `onto status` correctly reports `verify — skeleton: missing verification.md` (verify's deliverable, produced during verify).

## Coherence

- ontostate helpers ↔ `onto advance` compose correctly; advance registered once on the root (init/new/status/version/advance).
- **Corrected gate semantics** (mid-build fix, opus-reviewed): the advance precondition checks `RequiredArtifacts(st.Phase)` (the CURRENT phase's cumulative deliverables) — advancing FROM a phase requires THAT phase's artifacts (leaving open needs only proposal+tasks; leaving design needs design.md; leaving build needs plan.md + all tasks checked; leaving verify needs verification.md). Delta spec, design doc, code, and tests all agree; no stale `RequiredArtifacts(next)` remains.
- **No-write-on-failure (CRITICAL invariant):** `ontostate.Save` is the last step; every gate/precondition/close-block failure returns before it; the recorded phase never changes on refusal (final review confirmed).
- **Dirty-worktree:** WARN normal / BLOCK close; undeterminable git BLOCKs close conservatively; no false-clean parse.
- **Isolation & security:** `internal/ontocli`/`internal/ontostate` import NONE of homonto's `internal/{cli,engine,config,adapter,catalog}`; git invoked as an arg-vector (no shell injection); `--dir` is the caller's own arg; change name hardened by `validChangeName`.
- Final whole-branch review (opus): **READY TO MERGE**, 0 Critical / 0 Important.

## Scope boundary (honest)

onto binary #3b (second sub-increment of the onto workflow engine). Adds `onto advance` + phase helpers to `onto-binary`. NOT included / not claimed: dependency resolution + archive/close side effects (#3c), `onto doctor` (#4), dual-binary release packaging (#5). Dual-binary gate NOT met. Docs state this accurately.

## Accepted follow-ups (SUGGESTION)

- OF-g1: `TasksAllChecked` checkbox detection is prefix-anchored (embedded `- [ ]` in prose ignored) — spec-compliant.
- OF-g2: a test's open/bogus want-slice share identity (harmless).
- Task2 cosmetics: `TasksAllChecked` error not re-prefixed with `onto advance:`; undeterminable-git on a non-close advance proceeds silently (spec-conformant).

## Security

`onto advance` mutates only the `phase` field of an existing `onto-state.yaml` via atomic `Save`, gated on a real homonto workspace + current-phase deliverables + (for close) a clean worktree; refuses with no write on any failure. git is exec'd as an arg-vector. No new dependency. `onto` isolated from the projection pipeline.
