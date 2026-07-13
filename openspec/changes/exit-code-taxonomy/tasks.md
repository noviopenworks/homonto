# Tasks — exit-code-taxonomy
## 1. Opt-in exit-code taxonomy
- [x] cli.Execute(args) int + a testable exit-code sink; main.go uses it (errors -> 1).
- [x] plan --exit-code: 0 clean / 2 pending. status --exit-code: 0 clean / 2 pending / 3 drift.
      Default (no flag) unchanged. Tests: Execute returns the right code per state; helper tests.
## 2. Verify
- [x] go test ./internal/cli/... -race, vet, build, openspec validate --all green.
