## Why

ROADMAP E2 / finding F51: remote cache GC exists internally
(`Engine.GCRemoteCache`, `Cache.GC`) but there is no CLI to run it, so unreferenced
content-addressed cache entries can only accumulate. Apply deliberately does not GC
(keeping a repin fast/offline), so reclamation needs an explicit command.

## What Changes

- Add `homonto cache gc [--dry-run]`: reclaims content-addressed remote cache
  entries no remote lock references (via `Engine.GCRemoteCache`), printing what it
  removed (or would remove under `--dry-run`).

## Impact

- **Code:** `internal/cli/cache.go` (new), `internal/cli/root.go` (register) + test.
- **Spec:** `cli-commands` delta (cache gc command).
- **Out of scope:** the rest of E2 (JSON output, exit codes).
