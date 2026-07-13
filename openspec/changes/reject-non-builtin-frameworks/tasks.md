# Tasks — reject-non-builtin-frameworks
## 1. Reject non-builtin framework sources at load
- [ ] config.Load rejects a [frameworks.X] source that is not builtin: with a clear
      error. Test: a local: framework is rejected; a builtin: framework still loads.
## 2. Verify
- [ ] go test ./internal/config/... -race, vet, build, openspec validate --all green.
