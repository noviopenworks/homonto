Preset: tweak

# Proposal: expand-ci

## Why

CI runs only `go vet` and `go test` (NEXT_AGENT gap #7). It does not catch
unformatted code, an untidy `go.mod`, build breaks in non-test packages, data
races, a broken version stamp, or a broken CLI — all of which can land on main.

## What Changes

Expand `.github/workflows/ci.yml` to also run, on every push/PR:
- `gofmt -l` check (fail on unformatted files)
- `go mod tidy -diff` (fail on an untidy module)
- `go build ./...`
- `go test -race ./...`
- a **version-stamp smoke**: build with `-ldflags -X …cli.Version=…` and assert
  `homonto version` prints the stamped value
- a **CLI smoke**: run `homonto plan` on a minimal config

No source or spec change — CI configuration only.

## Capability Impact

- Untouched: no living spec requirement changes (CI is not specced; the existing
  `cli-commands` version-stamp requirement is exercised, not modified).

## Grounding

`.github/workflows/ci.yml` (vet+test only). Version stamping:
`internal/cli/root.go` `Version` var + `-ldflags -X …cli.Version`. All added
commands verified passing locally before writing the workflow.

## Impact

- Files: `.github/workflows/ci.yml` only.
- Risk: a too-strict step could red the pipeline. Mitigated by running every
  added command locally first (all green) and using the same Go 1.23 as CI.
