# Tasks — adapter-conformance-adoption-drift
## 1. Extend conformance: drift + malformed-doc safety
- [ ] Add to the shared suite for claude+opencode: out-of-band change -> ObserveHashes
      reports drift; re-Apply resets it; a pre-existing malformed tool doc does not
      panic Plan/Apply. Skip explicitly (with comment) any check an adapter can't meet.
## 2. Verify
- [ ] go test ./internal/adapter/... -race, vet, build, openspec validate --all green.
