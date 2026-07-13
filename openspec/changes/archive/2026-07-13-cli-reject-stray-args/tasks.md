# Tasks — cli-reject-stray-args
## 1. NoArgs on positional-free commands
- [x] plan/apply/status/doctor/import set cobra.NoArgs; init keeps MaximumNArgs(1).
- [x] Test: `homonto apply extra` (and one other) exits non-zero naming unexpected args.
## 2. Verify
- [x] `go test ./internal/cli/... -race`, vet, build, `openspec validate --all` green.
