# Tasks — transactional-remote-apply
## 1. Stage-then-swap (F8)
- [ ] Fetch+verify ALL declared remotes into staging before pruning/mutating any
      active content or the lock; a later failure leaves active content + lock
      unchanged. Test: a 2nd-remote failure rolls back / leaves first untouched.
## 2. Digest repin in plan + confirm (F6)
- [ ] A digest-only repin surfaces as a plan change and requires confirmation
      before remote mutation. Test: repin is not silently applied under "no changes".
## 3. Bounded git fetch (F27)
- [ ] Git fetch under a deadline; size/file guards at/**before** checkout. Test.
## 4. Quarantine revoked + doctor digest verify (F30)
- [ ] Revoked content deactivated; doctor verifies materialized digests vs lock. Test.
## 5. Cache re-hash on race (F26)
- [ ] Re-hash a cache-race winner before acceptance. Test.
## 6. Verification
- [ ] `go test ./internal/... -race`, vet, build, `openspec validate --all` green.
