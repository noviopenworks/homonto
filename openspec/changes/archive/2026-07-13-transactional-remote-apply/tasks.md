# Tasks — transactional-remote-apply
## 1. Stage-then-swap (F8)
- [x] Fetch+verify ALL declared remotes into staging before pruning/mutating any
      active content or the lock; a later failure leaves active content + lock
      unchanged. Test: a 2nd-remote failure rolls back / leaves first untouched.
## 2. Digest repin in plan + confirm (F6)
- [x] A digest-only repin surfaces as a plan change and requires confirmation
      before remote mutation. Test: repin is not silently applied under "no changes".
## 3. Bounded git fetch (F27)
- [x] Git fetch under a deadline; size/file guards at/**before** checkout. Test.
## 4. Quarantine revoked + doctor digest verify (F30)
- [x] Revoked content deactivated; doctor verifies materialized digests vs lock. Test.
## 5. Cache re-hash on race (F26)
- [x] Re-hash a cache-race winner before acceptance. Test.
## 6. Verification
- [x] `go test ./internal/... -race`, vet, build, `openspec validate --all` green.
