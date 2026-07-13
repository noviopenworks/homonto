# Verification — remote-frameworks (E1 remote framework resolution)

Full verification (full workflow + delta spec). PASS.

| # | Check | Result |
|---|-------|--------|
| 1 | tasks.md all `[x]` | PASS |
| 2 | Matches design (reuse remote.Resolver + local-overlay path; wiring only) | PASS |
| 3 | Delta scenarios (digest-pinned remote framework installs; mismatched digest aborts fail-closed) | PASS |
| 4 | proposal goals (framework sources completed: builtin/local/remote) | PASS |
| 5 | E2E gates: RemoteFrameworkSkillMaterialized + WrongDigestAborts | PASS |
| 6 | `go test ./... -race` (681) | PASS |
| 7 | vet, build, openspec validate --all (16/16) | PASS |
| 8 | Code review (standard, manual — SECURITY-sensitive) | PASS |

## Code review (standard, manual — security)
- **Verify-before-use**: `resolveRemoteFrameworks` calls `resolver.Resolve(ctx, src, pin)`, which fetches and verifies content against the pinned digest before returning the cache dir; the wrong-digest gate aborts here (Build fails). No new crypto/fetch/verify — reuses `internal/remote` verbatim.
- **Revocation fail-closed** *before* Resolve (`rev.Contains(pin)` → error), mirroring `materializeRemotes` (F30) — a revoked pin never serves from a warm cache.
- **Digest required**: `validateFrameworkResources` requires a valid digest for a remote framework and parses it at load; builtin/local still reject digest; any other source still fails. A remote framework with no/invalid digest fails at load.
- **Blast radius**: `materializeRemotes` (subagent lock/prune/repin/revocation) untouched; resolution runs at Build only for configs that declare remote frameworks (content-addressed cache → network only on first/changed pin; builtin/local/remote-subagent configs unchanged — 681 tests green).
- Independently re-ran build + both gates + full -race + vet after the delegated change; gate test untouched.

## Behavior / risk
Medium — wiring two tested subsystems; no new security code. Completes the framework-source trinity (builtin/local/remote). Remaining E1: `[compat].homonto`, capabilities, F38 — decision-gated/premature.
