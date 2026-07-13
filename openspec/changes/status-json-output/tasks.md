# Tasks — status-json-output
## 1. status --output json
- [x] `--output text|json` on status; json emits drift/pending/warnings as JSON; text unchanged.
      Test: json parses and carries the fields; an invalid --output value errors.
## 2. Verify
- [x] go test ./internal/cli/... -race, vet, build, openspec validate --all green.
