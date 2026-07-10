# Verification Report: onto-binary-foundation

**Date:** 2026-07-10
**Mode:** full (16 tasks, 1 delta capability, 18 changed files)
**Branch:** feature/20260710/onto-binary-foundation (base-ref 06e1420)

## Summary

| Dimension    | Status |
|--------------|--------|
| Completeness | 16/16 OpenSpec tasks ✅; 4/4 plan tasks ✅; all 4 spec requirements implemented |
| Correctness  | 4/4 requirements covered by code + tests; 16 tests in the two new packages, 0 failures |
| Coherence    | Follows design.md + Design Doc; isolated from homonto; phase set corrected to open\|design\|build\|verify\|close |

**Final assessment: All checks passed. Ready for archive.** 0 CRITICAL, 0 WARNING. Two Important review findings were fixed during build (Load names the file on malformed content; dead archive-skip branch removed). Four SUGGESTION-level follow-ups accepted (OF1/OF3/OF4/OF5).

## Completeness

- OpenSpec `tasks.md`: 16/16 checked (`grep -c '- [ ]'` → 0).
- All four ADDED requirements of `specs/onto-binary/spec.md` have implementation + tests (below).

## Correctness — requirement → implementation → test (fresh run 2026-07-10)

| Requirement | Implementation | Test evidence |
|---|---|---|
| Onto binary builds independently | `cmd/onto/main.go` (package main) | `go build ./cmd/onto` + `go build ./...` build both binaries; root `homonto`/`internal/cli` untouched (git diff empty) |
| Onto CLI root and version | `internal/ontocli/root.go` (`NewRootCmd`, `Version` ldflags var) | `TestNewRootCmdUse`, `TestVersionCommand`; `onto version` → `onto 0.1.0-dev` |
| onto-state.yaml model | `internal/ontostate/state.go` (Parse/Load/Validate/DerivePhase) | 9 tests incl. valid parse+derive, malformed-YAML error names "onto-state", **Load names the file path on malformed content** (fix), unknown-phase, empty-change, missing-file, no-panic on garbage |
| onto status read-only + config-independent | `internal/ontocli/status.go` | 7 tests incl. reports valid/invalid, **read-only tree snapshot (zero file writes)**, config-independence (no homonto.toml), archive-excluded, DerivePhase-error branch |

**Fresh gates:** `go build ./...` clean (both binaries); `go test ./... -count=1` → 0 FAIL (255 tests); `go test -race` on the new packages clean; `go vet ./...` clean; `gofmt -l .` empty; `go mod tidy` clean (yaml.v3 pinned). Behavior: `onto version` → `onto 0.1.0-dev`; `onto status --dir <ws>` → `c1: design`, exit 0.

## Coherence

- Follows `design.md`/Design Doc: `cmd/onto` + `internal/ontocli` (mirrors `internal/cli`) + `internal/ontostate`; `gopkg.in/yaml.v3` added, confined to `internal/ontostate`; `onto status` strictly read-only + config-independent.
- **Isolation:** the three new packages import NONE of homonto's `internal/cli`, `engine`, `config`, `catalog`, `adapter`; `main.go`/`internal/cli` unchanged (confirmed by final review + git diff).
- **Phase set correction** (design-phase Spec Patch): open|design|build|verify|close (onto workflow; terminal `close`), matching the `onto-*` skills and legacy `state.yaml` — not the comet dev `archive`.
- Final whole-branch review (opus): **READY TO MERGE**, 0 Critical / 0 Important.

## Scope boundary (honest)

This is change #1 of 5 for the onto binary — the FOUNDATION only. NOT included and explicitly not claimed: `onto init` (#2), phase-gate enforcement (#3), `onto doctor` (#4), dual-binary release packaging (#5). The dual-binary release gate is NOT met by this change. Docs (README, road-to-release, roadmap) state this accurately.

## Accepted follow-ups (non-blocking, SUGGESTION)

- OF1: `ontostate.Parse` belt-and-suspenders `recover()` (masks future non-yaml panics) — accept per the "never panic" contract.
- OF3: `onto status --dir` unvalidated (no `..`/symlink checks) — accept for a local read-only diagnostic; harden if ever fed untrusted input.
- OF4: status success line uses `State.Change`, invalid line uses dir name (brief-specified) — cosmetic if they diverge.
- OF5: read-only test snapshots files, not empty dirs — status creates no dirs; low.

## Security

`onto status` has zero write paths (only `os.ReadFile`/glob), reads no `homonto.toml`, constructs no config/engine. onto-state.yaml parsing is panic-safe (garbage/alias inputs tested); yaml.v3's built-in alias-expansion cap is intact. New dependency `gopkg.in/yaml.v3` is standard, `govulncheck`-covered, confined to one package.
