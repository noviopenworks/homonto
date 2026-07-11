# Verification Report: onto-skeleton

**Date:** 2026-07-10 · **Mode:** full · **Branch:** feature/20260710/onto-skeleton (base-ref 08834df)

## Summary

| Dimension | Status |
|---|---|
| Completeness | 12/12 OpenSpec tasks ✅; 4/4 plan tasks ✅; all 3 delta requirements implemented |
| Correctness | 3/3 requirements + 8 scenarios covered by code + tests; 22 ontocli + ontostate tests, 0 failures |
| Coherence | Follows design.md + Design Doc; isolated from homonto; onto binary #3a of the workflow engine |

**Final: All checks passed. Ready for archive.** 0 CRITICAL, 0 WARNING. One Important review finding (empty-vs-nil Deps round-trip) determined to be an inherent YAML/Go impedance with no production impact (binary uses nil Deps). Five SUGGESTION follow-ups accepted (OF-s1..s5).

## Correctness — requirement → implementation → test (fresh run 2026-07-10)

| Requirement | Implementation | Test/E2E evidence |
|---|---|---|
| onto-state.yaml writer | `ontostate.Marshal`/`Save` (atomic temp+rename) | round-trip `Parse(Marshal(s))`; Save→Load; parent-dir creation |
| onto new creates a change skeleton (gated, no-clobber, name-validated) | `ontocli.newCmd`/`runNew` + `validChangeName` | creates onto-state.yaml(open)+proposal+tasks exit 0; no-clobber byte-check; `../evil` rejected; gate-failure no writes; `created` YYYY-MM-DD. E2E: `onto new demo` → skeleton |
| phase-aware skeleton validation | `ontostate.RequiredArtifacts`/`ValidateSkeleton`; `onto status` note | status "skeleton ok" / "skeleton: missing <file>"; read-only tree snapshot. E2E: `demo: open — skeleton ok` |

**Fresh gates:** `go build ./...` clean (both binaries); `go test ./... -count=1` → 0 FAIL (276 tests); `go test -race` on the two packages clean; `go vet ./...` clean; `gofmt -l .` empty; `go mod tidy` clean (no new deps).

## Coherence

- ontostate writer ↔ `onto new` (uses Save) ↔ `onto status` (uses ValidateSkeleton) compose correctly; new/status/init/version registered once each on the root.
- **Isolation:** `internal/ontocli` + `internal/ontostate` + `cmd/onto` import NONE of homonto's `internal/{cli,engine,config,adapter,catalog}` (final review grep-confirmed).
- **Path-safety:** `validChangeName` (rejects `..`/separators/non-Base/non-kebab) runs before the one path-building join used for writes; a single kebab segment cannot escape `docs/changes/`. No-clobber refuses a pre-existing change before any write.
- Final whole-branch review (opus): **READY TO MERGE**, 0 Critical / 0 Important.

## Scope boundary (honest)

onto binary #3a (first sub-increment of the onto workflow engine, originally scoped as #3). Adds writer + `onto new` + skeleton validation to `onto-binary`. NOT included / not claimed: phase-transition gating (#3b), dependency resolution + archive/close rules (#3c), `onto doctor` (#4), dual-binary release packaging (#5). Dual-binary gate NOT met. Docs state this accurately.

## Accepted follow-ups (SUGGESTION)

- OF-s1: Deps `[]string{}` round-trips to nil (inherent YAML/Go; binary uses nil).
- OF-s2: Save error-cleanup branches untested; RequiredArtifacts only tested for "open"; `Save` uses a fixed `path+".tmp"` suffix (plan said random) — harmless for a single-shot CLI.
- OF-s3: `onto new` exists-check treats any Stat error (e.g. EACCES) as absent (a later MkdirAll would still fail — no clobber).
- OF-s4: partial-write mid-`onto new` leaves a half-populated changeDir (no rollback; matches `runInit`).
- OF-s5: status missing-artifact note is verbose (carries raw os.Stat error).

## Security

`onto new` writes only under a validated single-segment name joined to `--dir/docs/changes/`, gated on a real homonto workspace, never clobbering; `Save` is atomic. `onto status` is read-only. No new dependency. `onto` isolated from the projection pipeline.
