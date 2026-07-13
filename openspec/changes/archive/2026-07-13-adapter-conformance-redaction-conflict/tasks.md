# Tasks — adapter-conformance-redaction-conflict
## 1. Extend conformance: secret non-resolution + foreign-content safety
- [x] Add to the shared suite for claude+opencode: a secret reference is not resolved/
      leaked via ObserveHashes or on disk; foreign (unowned) on-disk content for a
      managed key is not silently clobbered/adopted. Skip explicitly (comment) if an
      adapter can't express a check.
## 2. Verify
- [x] go test ./internal/adapter/... -race, vet, build, openspec validate --all green.
