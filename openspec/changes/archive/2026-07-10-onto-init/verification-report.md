# Verification Report: onto-init

**Date:** 2026-07-10
**Mode:** full (13 changed files, 1 delta capability — modifies `onto-binary`)
**Branch:** feature/20260710/onto-init (base-ref 5bfd362)

## Summary

| Dimension    | Status |
|--------------|--------|
| Completeness | 8/8 OpenSpec tasks ✅; 3/3 plan tasks ✅; both delta requirements implemented |
| Correctness  | 2/2 requirements + 5 scenarios covered by code + tests; 14 ontocli tests, 0 failures |
| Coherence    | Follows design.md + Design Doc; isolated from homonto; gate-before-write; onto binary #2 of 5 |

**Final assessment: All checks passed. Ready for archive.** 0 CRITICAL, 0 WARNING. Two Important review findings fixed during build (per-task). Four SUGGESTION follow-ups accepted (OF-i1..i4).

## Correctness — requirement → implementation → test (fresh run 2026-07-10)

| Requirement | Implementation | Test evidence |
|---|---|---|
| onto init scaffolds the workflow layout (idempotent, no overwrite) | `internal/ontocli/init.go` `initCmd`/`runInit` — stat-then-`os.MkdirAll` for `docs/{changes,specs,adr,guides}`, created-vs-exists report | `TestInitCommand_ScaffoldsLayout`, `TestInitCommand_IsIdempotentAndNeverOverwrites` (byte-identity on a user file) |
| onto init requires the Homonto-managed framework install | `gate(root)` — homonto.toml + `[frameworks.onto]` + `.homonto/catalog/skills/onto/` (go-toml/v2, no config.Load) | `TestGate_*` (4 outcomes), `TestInitCommand_GateFailureCreatesNothing` (no docs/ written, non-zero exit) |

**Fresh gates:** `go build ./...` clean (both binaries); `go test ./... -count=1` → 0 FAIL (262 tests); `go test -race ./internal/ontocli` clean; `go vet ./...` clean; `gofmt -l .` empty; `go mod tidy` clean (no new deps). **E2E:** prepared workspace → `onto init` scaffolds `docs/{changes,specs,adr,guides}`, exit 0; empty dir → `error: onto init: no homonto.toml found in <dir>; run \`homonto init\` first`, exit 1, no docs written; second run idempotent (user file preserved).

## Coherence

- Follows design.md/Design Doc: gate → scaffold in `internal/ontocli/init.go`; gate uses `go-toml/v2` directly (no `config.Load`, no engine); `--dir` mirrors `onto status`; `initCmd` registered once on the root.
- **Isolation:** `internal/ontocli` + `cmd/onto` import NONE of homonto's `internal/{cli,engine,config,adapter,catalog}` (final review grep-confirmed). No change to `onto status`, `internal/ontostate`, or `homonto`.
- **Mutating-command safety:** gate-before-write is a structural early-return; only `os.MkdirAll` writes (never overwrites/truncates); the four scaffolded paths are fixed literals joined under `--dir` (no user-controlled segment → no write outside `docs/`); gate failure writes nothing.
- Final whole-branch review (opus): **READY TO MERGE**, 0 Critical / 0 Important.

## Scope boundary (honest)

onto binary #2 of 5. Adds `onto init` + framework gate to the `onto-binary`
capability. NOT included / not claimed: phase-gate enforcement + skeleton
create-validate (#3), `onto doctor` (#4), dual-binary release packaging (#5). The
dual-binary release gate is NOT met by this change. Docs state this accurately.

## Accepted follow-ups (non-blocking, SUGGESTION)

- OF-i1: gate malformed-TOML parse path wrapped but not unit-tested.
- OF-i2: gate-failure test had a weak RED (cobra unknown-command also err!=nil); GREEN exercises the real path.
- OF-i3: `docsLayout` literal duplicated in init.go and init_test.go (drift risk).
- OF-i4: theoretical stat/MkdirAll TOCTOU on the created-vs-exists report line (cosmetic; single-shot CLI).

## Security

`onto init` writes only via `os.MkdirAll(0o755)` on fixed `docs/*` paths under a caller-supplied `--dir`, gated on that root being a real homonto workspace; no write outside `docs/`, no overwrite, nothing on gate failure. Reads homonto.toml via go-toml/v2 (no new dependency). `onto` remains isolated from the projection pipeline.
