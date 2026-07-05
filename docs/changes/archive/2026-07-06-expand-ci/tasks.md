# Tasks: expand-ci

## 1. Build
- [x] 1.1 Expand `.github/workflows/ci.yml`: gofmt check, `go mod tidy -diff`,
      build, vet, test, `-race`, version-stamp smoke, CLI smoke.

## 2. Validate
- [x] 2.1 Every added command run locally on Go 1.23 and passes (gofmt clean,
      tidy no diff, build/vet/test/race green, version stamp prints the stamped
      value, `plan` smoke works). YAML is well-formed.
