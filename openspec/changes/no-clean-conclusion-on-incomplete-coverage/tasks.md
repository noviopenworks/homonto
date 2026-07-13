# Tasks — no-clean-conclusion-on-incomplete-coverage
## 1. Shared coverage helper + wire plan/status
- [ ] Add coverageComplete(warnings) error (non-nil when warnings present). plan/status
      return it instead of the clean conclusion when warnings exist; apply reuses it.
- [ ] Test the helper (nil when empty; error naming warnings otherwise).
## 2. Verify
- [ ] `go test ./internal/cli/... -race`, vet, build, `openspec validate --all` green.
