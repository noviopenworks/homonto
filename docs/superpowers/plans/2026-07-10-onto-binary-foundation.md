---
change: onto-binary-foundation
design-doc: docs/superpowers/specs/2026-07-10-onto-binary-foundation-design.md
base-ref: 06e14209a5145216adaafbb3bb8aa516f4ccce96
---

# Onto Binary Foundation Implementation Plan

> **For agentic workers:** implement task-by-task with TDD. Steps use checkbox syntax.

## Global Constraints (from Design Doc + delta spec)

- Module path `github.com/noviopenworks/homonto`. Two binaries: root `main.go` →
  `homonto` (UNCHANGED); new `cmd/onto/main.go` → `onto`.
- onto workflow phases: **open|design|build|verify|close** (terminal `close`; NOT
  the comet `archive`).
- `onto status` is strictly READ-ONLY and config-independent: no `homonto.toml`,
  no engine/config construction, zero writes.
- `internal/ontostate` is the only importer of `gopkg.in/yaml.v3`.
- Mirror `internal/cli/root.go` (`NewRootCmd`, `Version` ldflags var) and root
  `main.go` structure for `internal/ontocli` / `cmd/onto/main.go`.
- Verification gates: `go build ./...`, `go test ./... -count=1`, `go test -race
  ./...`, `go vet ./...`, `gofmt -l .`, `go mod tidy` clean.
- Additive only: do NOT modify `internal/cli`, `main.go`, adapters, engine,
  config, or catalog.

## Task 1: `onto-state.yaml` model (`internal/ontostate`)

**Files:** create `internal/ontostate/state.go`, `internal/ontostate/state_test.go`; modify `go.mod`/`go.sum`.

- [x] 1.1 `go get gopkg.in/yaml.v3` then `go mod tidy`; confirm it appears in go.mod require and `go mod tidy` is clean (no diff)
- [x] 1.2 Write failing tests first in `state_test.go`: valid parse+derive (phase build); malformed-YAML error mentions "onto-state"; unknown-phase → validate error; empty-change → validate error; missing-file `Load` error; no panic on `Parse([]byte("\x00garbage"))`
- [x] 1.3 Run tests → RED (package has no State type yet / build failure)
- [x] 1.4 Implement `State` struct (fields: Change, Workflow, Phase, Created, BaseRef, Deps, Archived — yaml tags per Design D3), `Parse`, `Load` (wrap os error naming the path), `Validate` (Change non-empty; Phase ∈ open|design|build|verify|close), `DerivePhase` (Validate then return Phase)
- [x] 1.5 Run tests → GREEN; `gofmt -l internal/ontostate/`, `go vet ./internal/ontostate/` clean
- [x] 1.6 Commit: `feat(ontostate): onto-state.yaml model (parse/validate/derive-phase)`

## Task 2: onto binary + CLI root (`internal/ontocli`, `cmd/onto`)

**Files:** create `internal/ontocli/root.go`, `internal/ontocli/root_test.go`, `cmd/onto/main.go`.

- [x] 2.1 Write failing test in `root_test.go`: `NewRootCmd()` returns a cmd with Use "onto"; executing `version` prints `onto <Version>` (capture cmd output)
- [x] 2.2 Run → RED
- [x] 2.3 Implement `internal/ontocli/root.go`: `var Version = "0.1.0-dev"` (doc-comment noting ldflags stamping to `…/internal/ontocli.Version`); `NewRootCmd()` (Use:"onto", Short, Version, SilenceUsage/Errors) + `version` subcommand printing `onto %s`
- [x] 2.4 Create `cmd/onto/main.go` (`package main`) mirroring root `main.go`: `ontocli.NewRootCmd().Execute()`, print `error:` to stderr, `os.Exit(1)`
- [x] 2.5 Run → GREEN; `go build ./cmd/onto` produces onto; `go build ./...` builds both; gofmt/vet clean
- [x] 2.6 Commit: `feat(onto): cmd/onto binary + ontocli root and version`

## Task 3: `onto status` (read-only, config-independent)

**Files:** create `internal/ontocli/status.go`, `internal/ontocli/status_test.go`.

- [x] 3.1 Write failing tests in `status_test.go`: over a temp workspace with `docs/changes/<a>/onto-state.yaml` (valid, phase build) and `docs/changes/<b>/onto-state.yaml` (malformed) — running status prints a phase line for `<a>` and an "invalid" line for `<b>`, exits 0; snapshot the temp tree before/after and assert NO file created/modified/removed; a run with no `homonto.toml` present succeeds
- [x] 3.2 Run → RED
- [x] 3.3 Implement `statusCmd()`: resolve workspace root (`--dir` flag default `.`), glob `docs/changes/*/onto-state.yaml` (skip `docs/changes/archive/`), load each via `internal/ontostate`, print `"<change>: <phase>"` or `"<change>: invalid (<reason>)"`; never abort on one bad file; construct no config/engine; perform zero writes; exit 0 on clean read
- [x] 3.4 Register `statusCmd()` on the onto root in `NewRootCmd()`
- [x] 3.5 Run → GREEN; gofmt/vet clean
- [x] 3.6 Commit: `feat(onto): read-only config-independent 'onto status'`

## Task 4: Regression and docs

- [ ] 4.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` then `git diff --exit-code go.mod go.sum` (clean)
- [ ] 4.2 Update `docs/road-to-release.md` and `docs/roadmap.md`: note the onto binary FOUNDATION landed (second binary builds, `onto-state.yaml` model, read-only `onto status`); onto init/gates/doctor + dual-binary packaging remain (changes #2–#5). Do not over-claim (init/doctor/gates NOT done)
- [ ] 4.3 Commit: `docs: record onto binary foundation landed`

## Self-Review

- No modification to `main.go`, `internal/cli`, adapters, engine, config, catalog.
- `onto status` writes nothing (before/after tree snapshot proves it).
- Only `internal/ontostate` imports yaml.v3; `go mod tidy` clean.
- Phase set is open|design|build|verify|close everywhere.
