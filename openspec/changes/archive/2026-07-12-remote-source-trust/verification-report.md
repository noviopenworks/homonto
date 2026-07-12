# Verification Report — remote-source-trust

- **Date:** 2026-07-12
- **Change:** remote-source-trust (roadmap item 10 — Remote Trust)
- **Mode:** full
- **Base-ref:** `6c52e7b` → HEAD (`feature/20260712/remote-source-trust`)

## Result: PASS

### 1. Tasks complete

All 9 task groups (24 sub-tasks) in `tasks.md` are `[x]`; the Superpowers plan
tasks are all checked. `grep -c '- \[ \]' tasks.md` → 0.

### 2. Implementation matches design

Implementation follows `openspec/changes/remote-source-trust/design.md` and the
Design Doc `technical-design.md`:
package layout (`internal/remote/{digest,locator,extract,canonical,fetch,cache,
revoke,verify,lock}.go`), the verify-before-mutate ordering, canonical digest,
content-addressed cache, and the resolver seam are all as designed. One design
refinement landed during build (caught by testing): apply does **not** auto-GC
so a revert rolls back from cache; GC is the explicit `Engine.GCRemoteCache`.

### 3. Spec scenario coverage

| Delta spec requirement | Verifying test |
|---|---|
| Remote source declaration + mandatory pinning | `TestParseDigest`, `TestRemoteSourceRequiresDigest` |
| Verify-before-mutate resolution | `TestResolveHappyPath`, `TestResolvePinMismatchFailsClosed`, `TestResolveRevokedFailsClosed` |
| Fail-closed archive validation | `TestValidateTarFailsClosed`, `TestValidateTarGzBombBoundedByTotal` |
| Content-addressed cache + offline | `TestCachePutHasDir`, `TestResolveOfflineFromCache`, `TestResolveCachedRejectsTamperedCache` |
| Remote lockfile records provenance | `TestLockRoundTrip`, `TestLockDiffStable` |
| Rollback / removal / GC | `TestRemoteSubagentRollbackAndRevocation`, `TestRemoteSubagentGCReclaimsAfterPrune`, `TestRemoteSubagentEndToEnd` (prune) |
| config-model: remote form + digest | `TestRemoteSourceRequiresDigest`, `TestRemoteRejectedForNonSubagentKinds` |
| apply-pipeline: remote routing + fail-closed abort | `TestRemoteSubagentEndToEnd`, `TestRemoteSubagentPinMismatchAbortsApply` |

### 4. proposal.md goals satisfied

A remote install is pinned (required sha256), verified (canonical digest match),
cacheable/offline, reproducible (diff-stable lock), revocable, and removable;
malformed/tampered/revoked content fails closed before any mutation — enforced by
the malicious-fixture suite. Non-goal honored: no automatic remote updates.

### 5. Security / review

High-effort workflow code review completed; all 5 correctness/security findings
fixed (`b044bc1`): https→http redirect downgrade, non-shallow git clone,
digest-only repin not re-fetched, remote accepted for non-subagent kinds,
warm-cache not re-verified. No hardcoded secrets. Two negligible nits accepted
(recorded in tasks.md).

### 6. Gate evidence

`scripts/gate.sh` → govulncheck clean under pinned `go1.26.5`; `go test -race
./...` → 22 packages green; dual-binary Docker E2E → ALL SUITES PASS;
`openspec validate remote-source-trust` → valid.

### Scope note (not a gap)

The trust engine is generic; apply-time wiring landed for the **subagent**
resource kind. Remote is now explicitly rejected for skills/commands/frameworks
at load (fail-closed) rather than silently dangling — a mechanical follow-up to
extend, not new trust design.
