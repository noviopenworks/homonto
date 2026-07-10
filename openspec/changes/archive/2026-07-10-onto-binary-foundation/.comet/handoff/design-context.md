# Comet Design Handoff

- Change: onto-binary-foundation
- Phase: design
- Mode: full
- Context hash: 7b107df2caf6bebd4b9aebd19f36a547fe0ffb9bbca1e48f17b396a3e4690662

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-binary-foundation/proposal.md

- Source: openspec/changes/onto-binary-foundation/proposal.md
- Lines: 1-73
- SHA256: f0db658fc8f7cc4e11d3161186bae44ba6dff8dd1d12567c992779ce4b062f11

```md
## Why

The v0.1.0-rc.1 release gate is dual-binary: `homonto` (the deterministic
config projector) plus `onto` (the managed spec-driven workflow operator). Today
only `homonto` exists in source (`main.go` at the repo root); there is no `onto`
binary. `onto` is the last release blocker
(`docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`). The full
`onto` binary is large, so it is split into five changes (see below); this
change delivers the **foundation increment** everything else builds on: the
second binary exists and can inspect an existing `onto` workspace read-only.

### Onto binary decomposition (this change is #1)

1. **onto-binary-foundation** (this change) — second `package main`, Cobra root,
   the `onto-state.yaml` model (parse/write/validate/derive phase), and
   `onto status` (read-only, works without config).
2. onto-init — `onto init` scaffolds `docs/{changes,specs,adr,guides}`, gated on
   `[frameworks.onto]` being applied through Homonto. (depends on #1)
3. onto-phase-gates — skeleton create/validate, structural-invariant enforcement,
   valid-gate-only transitions, dependency resolution, archive/close and
   dirty-worktree rules. (depends on #1, #2)
4. onto-doctor — `onto doctor` workflow health. (depends on #1, #3)
5. dual-binary-release-packaging — release CI builds and publishes both binaries
   with a shared `SHA256SUMS`; install docs. (depends on #1)

## What Changes

- Add a second `package main` at `cmd/onto/` that builds the `onto` binary via
  `go build ./cmd/onto` (and is installable with `go install .../cmd/onto`),
  leaving the existing root `homonto` binary untouched.
- Add a Cobra root command `onto` mirroring `homonto`'s CLI construction style
  (`internal/cli`), stamped with a `version` the same way `homonto version` is.
- Add an `onto-state.yaml` model in a new `internal/ontostate` package:
  parse/serialize the change state file, validate its structure, and derive the
  current phase from its fields. The file is named **`onto-state.yaml`** (no
  migration/back-compat for the legacy `state.yaml` name — pre-release).
- Add `onto status`: a **read-only degraded** command that inspects an existing
  `docs/` workspace and its `onto-state.yaml` files WITHOUT requiring
  `homonto.toml` or `[frameworks.onto]`, for diagnostics and recovery. It reports
  each change's derived phase and flags missing/invalid state; it never writes.
- This change does NOT add mutating commands (`init`, phase transitions,
  `doctor`), the docs scaffolder, or release packaging — those are changes #2–#5.

## Capabilities

### New Capabilities

- `onto-binary`: the managed `onto` workflow-operator binary. This change
  introduces the capability's foundation — the binary and CLI root, the
  `onto-state.yaml` state model (parse/validate/derive-phase), and the read-only
  `onto status` command. Later changes (#2–#4) extend this same capability with
  `init`, phase-gate enforcement, and `doctor`.

### Modified Capabilities

None. This is a net-new binary and capability; it does not change `homonto`'s
config-projection behavior or any existing spec.

## Impact

- New `cmd/onto/main.go` (second `package main`); the root `homonto` binary and
  `main.go` are unchanged.
- New `internal/ontostate/` (state model: types, parse, serialize, validate,
  derive-phase) with tests.
- New `internal/cli` entry for the `onto` root + `status` subcommand (either a
  new file in `internal/cli` or a sibling package; follow the existing
  `NewRootCmd` pattern).
- No change to `internal/adapter`, `internal/engine`, `internal/config`, or the
  catalog — `onto` is independent of the projection pipeline in this increment.
- Release packaging still ships only `homonto` after this change; dual-binary
  packaging is change #5.
- Advances the roadmap's sole remaining Immediate Next Work item (the `onto`
  binary) by landing its foundation.

```

## openspec/changes/onto-binary-foundation/design.md

- Source: openspec/changes/onto-binary-foundation/design.md
- Lines: 1-91
- SHA256: 99d37f35d356280cf10d4908b700c63667e91e0994bc10914ca909b41ade36cf

```md
## Context

The dual-binary release needs an `onto` binary beside `homonto`. Today only
`homonto` exists (`main.go` → `internal/cli.NewRootCmd`). This change lands the
foundation of the `onto-binary` capability: the second binary, its CLI root, the
`onto-state.yaml` model, and the read-only `onto status`. Mutating commands
(`init`, phase gates, `doctor`) and release packaging are separate later changes
(#2–#5 in the proposal). `onto` is a product feature (it operates the onto
workflow for homonto's users); it is independent of how this repo is developed
(comet) and of `homonto`'s config-projection pipeline.

## Goals / Non-Goals

**Goals**

- `go build ./cmd/onto` produces an `onto` binary; `go build ./...` builds both;
  root `homonto` untouched.
- `onto` Cobra root + `version` (ldflags-stampable), mirroring `internal/cli`.
- `internal/ontostate`: parse/validate `onto-state.yaml`, derive phase.
- `onto status`: read-only, config-independent workspace inspection.

**Non-Goals**

- `onto init` / docs scaffolding (change #2), phase-gate enforcement /
  skeleton create-validate (change #3), `onto doctor` (change #4), dual-binary
  release packaging (change #5).
- Any change to `homonto`'s behavior, `internal/cli`, adapters, engine, config,
  or the catalog.
- Migration from the legacy `state.yaml` name.

## Decisions

**D1 — Binary layout: `cmd/onto/main.go` + `internal/ontocli`.** Add a second
`package main` under `cmd/onto/` (Go's conventional multi-binary layout) whose
`main` calls `ontocli.NewRootCmd().Execute()`, exactly as root `main.go` calls
`cli.NewRootCmd()`. A new `internal/ontocli` package holds the `onto` root,
`Version` var, and subcommands, kept separate from `internal/cli` (the `homonto`
CLI) so the two binaries share nothing that could couple their flags/version.
Alternative (one package main with a mode flag) rejected — the design mandates
two distinct binaries.

**D2 — YAML parsing for `onto-state.yaml` (key dependency decision).** The repo
currently has NO YAML dependency (config is TOML via `go-toml/v2`); the state
file is mandated as `onto-state.yaml`. Decision: add `gopkg.in/yaml.v3` as the
single new dependency for the `onto` state model. Rationale: it is the de-facto
standard, well-audited, and covered by `govulncheck` in CI; hand-rolling a YAML
parser for a structured state file (phase + gate/dependency records to come in
#3) is error-prone. This is called out explicitly because the project prizes a
tiny dependency set — the trade-off is accepted for a real YAML file. (The Design
Doc will confirm the exact struct tags and the minimal field set.)

**D3 — Minimal foundation state schema; derive = validated read.** For this
increment `onto-state.yaml` carries at least a change identifier and a `phase`
field from the onto workflow phase set open → design → build → verify → close
(terminal `close`, matching the `onto-*` skills and legacy `state.yaml`; NOT the
comet dev terminal `archive`). "Derive phase" validates
the recorded phase is a known value and returns it. Full artifact-based phase
derivation and gate consistency are change #3/#4 concerns; the model is
structured so those extend it without a rewrite.

**D4 — `onto status` is strictly read-only and degraded-safe.** It discovers
changes by walking `docs/changes/*/onto-state.yaml`, loads each via
`internal/ontostate`, and prints a per-change phase line; unreadable/malformed
state is reported as invalid, not fatal to the whole run. It never constructs the
homonto config/engine and never writes — satisfying the design's "read-only
degraded exception" that works without `homonto.toml` or `[frameworks.onto]`.

## Risks / Trade-offs

- **New YAML dependency** → Mitigation: single well-known module, `govulncheck`
  in CI, confined to the `onto` state model.
- **onto-state.yaml schema churn across #2–#4** → Mitigation: keep the
  foundation schema minimal and additive; later changes add fields (gates,
  deps), not rewrite existing ones.
- **CLI duplication between `internal/cli` and `internal/ontocli`** → Accepted:
  the two binaries are intentionally decoupled; shared helpers can be extracted
  later only if a real need appears.

## Migration Plan

Purely additive: new `cmd/onto/`, `internal/ontocli`, `internal/ontostate`, and
one new go.mod dependency. No existing binary or package changes behavior.
Rollback is removing the new packages and the dependency.

## Open Questions

- Exact `onto-state.yaml` field set and struct tags (confirmed in the Design
  phase against `docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`
  and the legacy `state.yaml` shape for reference only).
- Whether `onto status` output should be plain lines or also offer `--json`
  (default: plain for the foundation; `--json` can come with `doctor` in #4).

```

## openspec/changes/onto-binary-foundation/tasks.md

- Source: openspec/changes/onto-binary-foundation/tasks.md
- Lines: 1-27
- SHA256: 7eba7d1532b908f150cb81639130b2d9c6efe6e02b3dbdb64d21346203db7e5d

```md
## 1. onto-state.yaml model (`internal/ontostate`)

- [ ] 1.1 Add `gopkg.in/yaml.v3` to go.mod (`go get gopkg.in/yaml.v3`); run `go mod tidy`
- [ ] 1.2 Define the `State` struct (change id/name, `phase`, minimal gate fields) with yaml tags in `internal/ontostate/state.go`
- [ ] 1.3 Implement `Parse([]byte) (State, error)` / `Load(path)` — unmarshal + wrap YAML/os errors with the file name; never panic
- [ ] 1.4 Implement `Validate()` (phase is one of open|design|build|verify|close — the onto workflow phase set, matching the onto-* skills and legacy state.yaml) and `DerivePhase() (string, error)` (validated recorded phase)
- [ ] 1.5 Unit tests: valid parse+derive, malformed-YAML error names the file, unknown-phase error, missing-file error

## 2. onto binary + CLI root (`cmd/onto`, `internal/ontocli`)

- [ ] 2.1 Create `internal/ontocli/root.go`: `Version` var (ldflags-stampable) + `NewRootCmd()` (Use "onto", SilenceUsage/Errors) + `version` subcommand, mirroring `internal/cli/root.go`
- [ ] 2.2 Create `cmd/onto/main.go` (`package main`) calling `ontocli.NewRootCmd().Execute()`, mirroring root `main.go`
- [ ] 2.3 Verify `go build ./cmd/onto` produces the binary and `go build ./...` still builds `homonto`
- [ ] 2.4 Test: `onto version` prints `onto <Version>`; a stamped `-ldflags -X ...Version=` value is reflected (build-tag or ldflags test, or a unit test on the version command output)

## 3. `onto status` (read-only, config-independent)

- [ ] 3.1 Implement `statusCmd()` in `internal/ontocli`: walk `docs/changes/*/onto-state.yaml`, load each via `internal/ontostate`, print a per-change phase line; report unreadable/malformed changes as invalid without aborting the run
- [ ] 3.2 Register `statusCmd()` on the onto root; ensure it never constructs the homonto config/engine and never writes
- [ ] 3.3 Tests: status over a temp `docs/changes/` with a valid change (phase reported) and an invalid one (flagged), asserting exit 0 and that no file was created/modified/removed (read-only)
- [ ] 3.4 Test: status works with no `homonto.toml` present (degraded/config-independent)

## 4. Regression and docs

- [ ] 4.1 Full regression: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `gofmt -l .`, `go mod tidy -diff` (or `go mod verify`)
- [ ] 4.2 Update `docs/road-to-release.md` / `docs/roadmap.md` to note the onto binary foundation (binary + state model + `onto status`) has landed; onto init/gates/doctor/packaging remain (changes #2–#5)
- [ ] 4.3 Commit all changes

```

## openspec/changes/onto-binary-foundation/specs/onto-binary/spec.md

- Source: openspec/changes/onto-binary-foundation/specs/onto-binary/spec.md
- Lines: 1-81
- SHA256: 401d8a41f4b8f5c9b3c20753d56901944550ed09cd64d6bde48e010bf74a126e

```md
## ADDED Requirements

### Requirement: Onto binary builds independently of homonto

The repository SHALL build a second binary `onto` from a dedicated
`package main` at `cmd/onto/`, via `go build ./cmd/onto` and installable with
`go install github.com/noviopenworks/homonto/cmd/onto`. The existing root
`homonto` binary (built from `main.go`) SHALL be unchanged, and `go build ./...`
SHALL build both.

#### Scenario: onto binary compiles from its own package main

- **GIVEN** the repository at a clean checkout
- **WHEN** `go build ./cmd/onto` runs
- **THEN** it produces an `onto` executable, and `go build ./...` still builds the `homonto` binary unchanged

### Requirement: Onto CLI root and version

The `onto` binary SHALL expose a Cobra root command `onto` constructed in the
same style as `homonto`'s `internal/cli.NewRootCmd`, with a `version` subcommand
that prints the build version. The version SHALL be a package-level variable
stampable at release time via `-ldflags "-X …Version=<tag>"`, mirroring how
`homonto`'s version is stamped.

#### Scenario: onto version prints the stamped version

- **WHEN** `onto version` runs
- **THEN** it prints `onto <version>` and exits 0
- **AND** a release build with `-ldflags "-X …Version=v0.1.0-rc.1"` prints that tag

### Requirement: onto-state.yaml change-state model

The `onto` binary SHALL read and validate a per-change state file named
`onto-state.yaml` through a dedicated state package. The model SHALL parse the
file into a typed structure, validate its structural fields, and derive the
current workflow phase from those fields. The file name is exactly
`onto-state.yaml`; there SHALL be no migration or back-compatibility layer for
the legacy `state.yaml` name (pre-release). Parsing an invalid or malformed
`onto-state.yaml` SHALL return a clear error identifying the file, not a panic.

The recognized workflow phases are `open`, `design`, `build`, `verify`, `close`
(the onto workflow phase set, matching the `onto-*` skills and the legacy
`state.yaml`), with `close` as the terminal phase.

#### Scenario: parse and derive phase from a valid onto-state.yaml

- **GIVEN** a valid `onto-state.yaml` recording a change's phase (one of open|design|build|verify|close) and gate fields
- **WHEN** the state model loads it
- **THEN** it returns the typed state and the derived phase without error

#### Scenario: malformed onto-state.yaml reports a clear error

- **GIVEN** an `onto-state.yaml` that is not valid YAML or is missing required fields
- **WHEN** the state model loads it
- **THEN** it returns an error naming the file and the problem, and does not panic

### Requirement: onto status is read-only and config-independent

`onto status` SHALL be a read-only diagnostic command that inspects an existing
`docs/` workspace and its `onto-state.yaml` files WITHOUT requiring a
`homonto.toml` file or a declared `[frameworks.onto]` entry (the read-only
degraded exception). It SHALL report each discovered change's derived phase and
flag any change whose state file is missing or invalid. `onto status` SHALL NOT
create, modify, or delete any file.

#### Scenario: status inspects a workspace without config

- **GIVEN** a project with `docs/changes/<name>/onto-state.yaml` but no `homonto.toml` and no `[frameworks.onto]`
- **WHEN** `onto status` runs
- **THEN** it reports each change's derived phase and exits 0 without writing any file

#### Scenario: status flags an invalid state file

- **GIVEN** a change whose `onto-state.yaml` is missing or malformed
- **WHEN** `onto status` runs
- **THEN** it reports that change as invalid/unreadable and still does not write any file

#### Scenario: status leaves the worktree untouched

- **WHEN** `onto status` runs against any workspace
- **THEN** no file under `docs/` or elsewhere is created, modified, or removed (read-only)

```
