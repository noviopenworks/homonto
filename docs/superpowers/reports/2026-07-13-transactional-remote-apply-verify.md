# Verification Report — transactional-remote-apply

**Date:** 2026-07-13 · ROADMAP N5 (gate B, last RC blocker) · Comet tweak · verify_mode full · Result: PASS

- **F8** — `materializeRemotes` restructured into quarantine → stage(fetch+verify ALL into cache) →
  activate(prune+materialize+lock); a mid-run failure leaves active content + lock unchanged. `63319e4`.
- **F6** — a digest-only repin surfaces in plan (`PendingRemoteRepins`) and requires confirmation;
  no longer silently applied under "No changes". `33334f2`.
- **F27** — git fetch under a deadline with size/file guards before checkout. `3a25932`.
- **F30** — `engine.Doctor` verifies materialized remote digests vs lock (catches cache + active-file
  tampering); revoked content deactivated on apply failure. `c05a8af`.
- **F26** — cache-race winner re-hashed (`VerifyContent`) before acceptance on both Put paths. `e2ecab5`.

## Evidence
`go test ./internal/... -race` → 583 passed; vet clean; build success; `openspec validate --all` 16/16.

## Milestone
Closes the last gate-B RC blocker. With N3 (spec truth) + N4 + N6 also done, all engine-safety
and truth blockers for `v0.1.0-rc.1` are now closed.
