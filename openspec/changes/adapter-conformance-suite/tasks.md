# Tasks — adapter-conformance-suite
## 1. Shared conformance harness (first slice)
- [x] A table-driven test over claude + opencode asserting: Plan→creates,
      Apply writes, ObserveHashes reports applied keys clean, second Plan is a
      no-op, an unmanaged file is preserved. Reuse existing per-adapter test setup.
## 2. Verify
- [ ] go test ./internal/adapter/... -race, vet, build, openspec validate --all green.
