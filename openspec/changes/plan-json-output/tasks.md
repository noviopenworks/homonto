# Tasks — plan-json-output
## 1. plan --output json
- [x] `--output text|json` on plan; json emits visible changes (action+key per tool),
      repins, warnings; NO Old/New values (secret safety); text unchanged; invalid rejected.
- [x] Test: json parses with the fields; invalid --output errors.
## 2. Verify
- [x] go test ./internal/cli/... -race, vet, build, openspec validate --all green.
