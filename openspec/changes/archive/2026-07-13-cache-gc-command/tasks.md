# Tasks — cache-gc-command
## 1. cache gc command
- [x] `homonto cache gc [--dry-run]` calls Engine.GCRemoteCache(dryRun) and reports
      reclaimed digests; cobra.NoArgs. Register under a `cache` parent.
- [x] Test: the command runs and reports (dry-run leaves the cache unchanged).
## 2. Verify
- [x] `go test ./internal/cli/... -race`, vet, build, `openspec validate --all` green.
