# Tasks — config-load-phases

## 1. Extract phase functions
- [ ] Extract decode/migrate/normalize/validate from config.Load in the same
      order with no behavior change; Load calls them in sequence. Config suite
      green unchanged; optionally a focused test per phase.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      config load/validation tests pass unchanged.
