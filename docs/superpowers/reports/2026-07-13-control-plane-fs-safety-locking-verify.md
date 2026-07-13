# Verification Report — control-plane-fs-safety-locking

**Date:** 2026-07-13 · ROADMAP N6 (gate B) · Comet tweak · verify_mode full · Result: PASS

- **F25** — `fsutil.WriteControlPlane` (no-follow: lstat final component, refuse a symlink,
  atomic temp+fsync+rename, preserve existing perms). Routed .homonto control-plane writes
  (state.json, remote.lock.json, catalog materialize, remote subagent files) through it;
  tool-config writes keep `WriteAtomic`. Commit 519438d.
- **F29** — `internal/applylock` O_EXCL lockfile at `.homonto/apply.lock`; apply acquires
  after Build/before Plan, defer Release; second concurrent apply fails fast. Commit 84f25da.
- **F31** — `remote.RedactLocator` strips URL userinfo + secret query tokens; applied at
  Lock.Set (lockfile) and every URL-bearing error (locator/fetch/verify, incl. git argv).
  Commit 2e0fbbf.
- Delta aligned to the shipped no-follow protection (destination-symlink refusal; full
  intermediate root-confinement noted as a possible future tightening, not required to close
  the planted-symlink attack).

## Evidence
`go test ./internal/... -race` → 577 passed; vet clean; build success; `openspec validate --all` 16/16.

## Out of scope
N5 (remote transactional staging) remains the last open gate-B RC blocker.
