## Why

Remote application is not transactional (ROADMAP N5, gate B; the last RC blocker):
- **F8:** `materializeRemotes` (`internal/engine/remote.go`) prunes de-declared
  content first, then fetches+materializes each remote in a loop; a later remote's
  failure leaves earlier content changed and the lockfile stale.
- **F6:** a digest-only repin leaves the projection plan empty, so `apply` prints
  "No changes" and mutates remote content with no confirmation (`apply.go:60-71`).
- **F27:** git fetch runs on `context.Background()` with size/file caps applied
  only AFTER checkout — a malicious pin can exhaust time/disk before validation.
- **F30:** revoked-but-still-declared content stays linked after a failed apply;
  `doctor` does not verify materialized digests against the lock.
- **F26:** on a cache rename race, the winning directory is accepted without
  re-hashing.

## What Changes

- **Stage before mutate:** all declared remotes are fetched + verified into a
  staging area BEFORE any active content or lock is pruned/mutated; if any remote
  fails, no active content or lock changed.
- **Digest in plan + confirm:** a digest-only repin appears as a change in `plan`
  and requires confirmation before `apply` mutates remote content.
- **Bounded git:** git fetch runs under a deadline with size/file guards applied
  before/at checkout, not after.
- **Quarantine revoked + doctor verifies digests:** revoked content is deactivated;
  `doctor` verifies each materialized remote digest against the lock.
- **Cache re-hash:** a cache-race winning directory is re-hashed before acceptance.

## Impact

- **Code:** `internal/engine/remote.go` (stage-then-swap), `internal/remote/`
  (fetch ctx+guards, cache re-hash, revocation), `internal/cli/{plan,apply}.go`
  (digest change surfaced + confirmed), `internal/cli/doctor.go` (digest verify) + tests.
- **Spec:** `remote-source-trust` + `apply-pipeline` deltas.
- **Out of scope:** N6 (done), N2 (onto semantic gates).
