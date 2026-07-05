# Verification Report: expand-ci

- **Date:** 2026-07-06
- **Mode:** light (why: `workflow: tweak`, CI-configuration-only change)
- **Range:** f509f08..HEAD on `tweak/20260706/expand-ci`
- **Result: pass**

## Scenario evidence

GitHub Actions can't run locally, so every workflow step was executed locally on
the same Go 1.23 the workflow pins, and the YAML was parsed.

| Step | Verdict | Evidence |
|---|---|---|
| YAML well-formed | pass | `python3 yaml.safe_load` → `YAML OK, steps: 10` (gofmt, tidy, vet, build, test, race, version smoke, cli smoke) |
| gofmt check | pass | `gofmt -l .` → empty |
| `go mod tidy -diff` | pass | → no diff |
| go build / vet / test | pass | build-ok, vet-ok, `go test ./...` 129 passed |
| `go test -race` | pass | race-ok (129) |
| version stamp smoke | pass | `go build -ldflags "-X …cli.Version=ci-smoke"` then `homonto version 2>&1 \| grep -q ci-smoke` → `STAMP OK` |
| cli smoke | pass | `homonto --config <empty> plan` → exit 0 (`No changes. Everything up to date.`) |

## Design conformance

Tweak — no design.md. `.github/workflows/ci.yml` gained exactly the proposed
steps; no source or spec changed.

## Adversarial pass

Skipped (light mode, optional): a CI-config change whose every command was
executed locally and passed, with the YAML parsed. Recorded skip.

## Regression

`go build ./...`, `go vet ./...`, `go test ./...` (129), `go test -race ./...`
(129), `gofmt -l .` (empty) — all green locally.

## Deviations

None for this change. Observation (out of scope, candidate follow-up): the
version-stamp step needs `2>&1` because `homonto version` — like all homonto
CLI output routed through cobra's `Print*` — writes to **stderr**, not stdout.
Harmless for the smoke; a future change could route user-facing output to
stdout for scriptability.
