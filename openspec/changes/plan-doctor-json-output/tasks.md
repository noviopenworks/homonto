# Tasks — plan-doctor-json-output (doctor slice)
## 1. doctor --output json
- [ ] `--output text|json` on doctor; json emits {"findings":[...]}; text unchanged; invalid rejected.
- [ ] Test: json parses with findings; invalid --output errors.
## 2. Verify
- [ ] go test ./internal/cli/... -race, vet, build, openspec validate --all green.
