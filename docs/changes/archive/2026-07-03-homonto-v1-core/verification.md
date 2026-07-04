# Verification Report: homonto-v1-core

**Date:** 2026-07-03
**Mode:** full · **Branch:** feature/20260703/homonto-v1-core
**Fresh evidence:** `go test -count=1 ./...` → **45 passed / 0 failed** (14 pkgs);
`go vet ./...` → clean; `go build ./...` → OK; binary smoke-tested
(init/plan/apply/status/doctor projecting into both tools, idempotent re-apply).

## Summary

| Dimension | Status |
|---|---|
| Completeness | 25/25 tasks `[x]`; 19/19 requirements implemented |
| Correctness | 34/34 scenarios covered by code; ~30 by automated tests, rest by CLI smoke |
| Coherence | Follows design.md + Design Doc (normalized model + adapters, hashed state) |

**No CRITICAL issues.** 2 WARNINGs, both previously reviewed and accepted for v1.

## Requirement → implementation map

**config-model** — `config.Load` + `MCP.TargetsOrAll` (`internal/config/config.go`);
tests `TestLoad`, `TestLoadMissingFile`. Tokens preserved verbatim. ✓

**apply-pipeline** — pure-dry-run `plan` (`internal/cli/plan.go`, `engine.Plan`);
two-phase confirmation-gated `apply` (`internal/engine/engine.go:60-90`,
`internal/cli/apply.go`); atomic temp+rename writes, state saved last; idempotent
re-apply (`e2e_test.go`); drift (`engine.Drift`, `status.go`). Tests:
`TestApplyAbortsBeforeWritingOnMissingSecret`, `TestEndToEndApplyIsIdempotent`,
`TestDriftDetectedAfterOutOfBandChange`. ✓

**secret-references** — `secret.Resolver` (`${pass}`/`${ENV}`), `secret.Hash`,
`secret.ResolveJSON` (leaf-level resolution); `state.Entry{Desired,Applied}`.
Plan/state never carry plaintext (redaction + hash). Tests: resolver/hash suites,
`TestStateHasNoPlaintextSecret`, `TestRenderedPlanNeverLeaksSecret`,
`TestSecretDriftPlanIsRedacted`, `TestSecretToLiteralTransitionRedacts`,
`TestSecretWithSpecialCharsDoesNotCorruptFile`. ✓

**tool-adapters** — surgical merge preserves unmanaged keys (both adapters);
Claude + OpenCode projection; symlinked owned content with conflict detection;
unparseable file skips one tool, others proceed (`engine.Warnings`). Tests:
`TestPlanThenApplyIsSurgicalAndIdempotent`, `TestOpenCodeProjectsMCPAndPreservesKeys`,
`TestOpenCodeLinksOwnedSkill`, `TestLinkConflictDoesNotClobber`,
`TestUnparseableToolFileDoesNotBlockOtherTool`. ✓

**cli-commands** — all 7 commands registered (`internal/cli/root.go`), `--config`
persistent flag, no add/remove mutators; `init` no-overwrite (`TestInitCreatesFilesAndSkipsExisting`);
`import` secret redaction + `--force` (`TestImportRedactsSecretsInEnv`); `doctor`
checks `pass` + tool config locations + owned skills
(`TestDoctorFlagsMissingSkillContent`, `TestDoctorChecksToolConfigLocations`). ✓

## Coherence

Implementation follows `design.md` decisions and the Design Doc: normalized
`Config` → per-tool adapters; reference-only secrets resolved after confirm,
all-at-once, two-phase; surgical merge; atomic writes / state last; symlinked
content; hashed-state idempotency (`{desired, sha256(resolved)}`) exactly as
specified in the Design Doc's O-1..O-7. Code-review Critical (unsafe secret
substitution) was fixed and regression-tested during the build review gate.

## Issues

### CRITICAL
None.

### WARNING (accepted for v1 — recorded in tasks.md)
1. **`import` covers Claude `mcpServers` only** — the `cli-commands` requirement
   text says "read the current Claude/OpenCode setup"; the implementation reads
   Claude MCP servers (with redaction). Accepted: v1 `import` is a best-effort
   bootstrap; the safety-critical parts (secret redaction, `--force` guard) are
   implemented and tested. Broader import is a documented follow-up.
2. **Some CLI-level scenarios are covered by code + smoke, not unit tests** —
   `apply` confirmation-decline path and `import --force` guard live in the cobra
   layer (`internal/cli`) and are exercised by manual smoke, not a dedicated test.
   Low risk; noted for follow-up test coverage.

### SUGGESTION (from review, recorded)
Cross-adapter partial-apply state reconciliation, drift-vs-pending-edit framing,
deleted-key reporting, dotted-name path escaping, CWD-relative `contentDir`.

## Assessment

**All checks passed. No critical issues. Ready for archive** (with 2 accepted
warnings and noted suggestions). 25/25 tasks complete, 19/19 requirements
implemented, 34 scenarios covered, full suite green.
