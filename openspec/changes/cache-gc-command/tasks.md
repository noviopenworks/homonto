# Tasks — cache-gc-command
## 1. cache gc command
- [ ] `homonto cache gc [--dry-run]` calls Engine.GCRemoteCache(dryRun) and reports
      reclaimed digests; cobra.NoArgs. Register under a `cache` parent.
- [ ] Test: the command runs and reports (dry-run leaves the cache unchanged).
## 2. Verify
- [ ] `go test ./internal/cli/... -race`, vet, build, `openspec validate --all` green.
