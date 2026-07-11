# Comet Design Handoff

- Change: onto-doctor
- Phase: design
- Mode: compact
- Context hash: db19208b84b5770064b63e5bb25638e70ced6b18d29c82a75e276fcb4ea27b2b

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-doctor/proposal.md

- Source: openspec/changes/onto-doctor/proposal.md
- Lines: 1-69
- SHA256: e2f7af2ee1b81c85795d408a374036cb46c32db636d86f4c0530e5f077e7c233

```md
## Why

The onto workflow engine is complete: `onto new` creates, `onto advance`
transitions, and `onto close` archives a change. But nothing checks whether an
existing onto workspace is *healthy* — that its docs layout is intact, its
`onto-state.yaml` files are valid, each change's phase matches the artifacts it
should have, its dependencies are resolved, and its archive layout is
well-formed. The dual-binary design defines `onto doctor` as the peer to
`homonto doctor`: `homonto doctor` checks installation/projection health, while
`onto doctor` checks workflow and project health. This change (#4 of the onto
binary work) adds `onto doctor`.

## What Changes

- Add `onto doctor [--dir]`: a strictly read-only, config-independent diagnostic
  that reports the health of an onto workspace and exits non-zero if any problem
  is found (so CI and smoke tests can gate on it). It writes nothing, never
  constructs a homonto config/engine, and never reads `homonto.toml` — onto
  stays isolated from homonto's projection pipeline. It runs regardless of
  whether the onto framework is installed (it is a diagnostic, not a mutation,
  so it is **not** behind the framework-install gate that `init`/`new`/`close`
  use); a missing docs layout is reported as a finding, not a refusal.
- Checks performed (each surfaced as an individual finding line):
  - **docs layout**: `docs/changes`, `docs/specs`, `docs/adr`, `docs/guides`
    each exist as directories (reusing `onto init`'s `docsLayout`).
  - **active change state validity**: for each `docs/changes/*/onto-state.yaml`,
    it loads and validates the model and derives the phase; a malformed or
    invalid file is a finding.
  - **phase-derivation-matches-artifacts**: for each valid change,
    `ontostate.ValidateSkeleton` confirms every artifact
    `RequiredArtifacts(phase)` names is present; a missing artifact is a
    finding.
  - **gates and dependencies consistent**: for each valid change,
    `ontostate.DepsResolved` reports any unresolved dependency as a finding; an
    active (non-archived-directory) change whose state already records
    `archived: true` is a finding (an archived change belongs under
    `docs/changes/archive/`, not in the active set).
  - **archive layout valid**: each `docs/changes/archive/*` entry is a directory
    containing a valid `onto-state.yaml` marked `archived: true`; a missing or
    invalid state file, or one not marked archived, is a finding.
- On a healthy workspace it prints a single `healthy` summary and exits 0. When
  findings exist it prints each finding, a count summary, and exits non-zero.
- Register `doctorCmd()` on the onto root. `onto` binary commands become:
  **advance / close / doctor / init / new / status / version**.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains the `onto doctor` read-only workflow/project health
  command (docs-layout, change-state-validity, phase-vs-artifacts,
  dependency-consistency, and archive-layout checks; non-zero exit on findings).

## Impact

- `internal/ontocli`: new `doctor.go` (`doctorCmd()`/`runDoctor()`), registered
  on the root; reuses `docsLayout` and the existing `ontostate` API (`Load`,
  `Validate`, `DerivePhase`, `ValidateSkeleton`, `RequiredArtifacts`,
  `DepsResolved`). No new exported `ontostate` helper is required.
- No new dependency. `onto` stays isolated from homonto's projection pipeline
  (`internal/{cli,engine,config,adapter,catalog}` remain unimported by
  `internal/ontocli`).
- Read-only: `onto doctor` performs zero writes, mirroring `onto status`.
- Completes #4 of the onto binary work; only dual-binary release packaging (#5)
  remains.

```

## openspec/changes/onto-doctor/design.md

- Source: openspec/changes/onto-doctor/design.md
- Lines: 1-99
- SHA256: 440053161a03a6110741424d1be707d2cbf058fe17c3606762793d6cb689d1ad

[TRUNCATED]

```md
## Context

`onto doctor` is the last onto workflow command (#4). It is a diagnostic peer to
`homonto doctor`: where `homonto doctor` checks installation/projection health,
`onto doctor` checks *workflow and project* health. All the primitives it needs
already exist in `internal/ontostate` (`Load`, `Validate`, `DerivePhase`,
`ValidateSkeleton`, `RequiredArtifacts`, `DepsResolved`) and in
`internal/ontocli` (`docsLayout`). This change is therefore an assembly of
existing read-only primitives into one report — no new `ontostate` API.

## Goals / Non-Goals

**Goals**
- One command that answers "is this onto workspace healthy?" with a clear list
  of findings and a CI-gateable exit code.
- Strictly read-only and config-independent, exactly like `onto status`.
- Reuse existing `ontostate`/`ontocli` primitives; add no exported helper.

**Non-Goals**
- Fixing problems (`doctor` diagnoses; it never mutates). No `--fix`.
- Installation/projection health — that is `homonto doctor`'s job.
- Configurable artifact roots outside `docs/` (a first-release non-goal).
- Checking `onto-state.yaml` content beyond what `Validate`/`DerivePhase`/
  `ValidateSkeleton`/`DepsResolved` already enforce.

## Decisions

### D1 — Ungated, read-only, `--dir` default `.`

`onto doctor` is modeled on `onto status`, not `onto init`: it is a read-only
diagnostic, so it is NOT behind the framework-install `gate`. A workspace with
no `homonto.toml` / no installed framework is a valid thing to diagnose — the
missing `docs/` layout is reported as a finding rather than causing a refusal.
It performs zero writes and imports none of homonto's projection packages
(`internal/{cli,engine,config,adapter,catalog}`), preserving onto's isolation.

### D2 — Findings model + exit code

`runDoctor` accumulates finding strings into a slice, printing each to stdout as
it goes (or collecting then printing — implementation detail). At the end:
- zero findings → print `healthy`, return `nil` (exit 0);
- ≥1 finding → print a count summary (e.g. `N problem(s) found`) and return a
  non-nil error so the process exits non-zero.

The root command sets `SilenceErrors`/`SilenceUsage`, and `cmd/onto/main.go`
prints `error: <err>` to stderr and exits 1, so returning a summary error gives
a clean non-zero exit without a cobra usage dump. Findings themselves print to
stdout; only the one-line summary rides the error to stderr.

### D3 — Check set and ordering

Checks run in a fixed, stable order so output is deterministic:
1. **docs layout** — stat each of `docsLayout`; a non-directory / missing path
   is a finding.
2. **active changes** — `filepath.Glob(root/docs/changes/*/onto-state.yaml)`
   (the single `*` cannot cross a separator, so it structurally excludes
   `docs/changes/archive/<name>/…`, exactly as `onto status` relies on). For
   each: `Load`→invalid finding; else `DerivePhase`→invalid finding; else
   `ValidateSkeleton`→missing-artifact finding; `DepsResolved(root, st.Deps)`
   →unresolved-dep finding; `st.Archived == true`→active-marked-archived
   finding.
3. **archive layout** — `filepath.Glob(root/docs/changes/archive/*)`; for each
   entry that is a directory, `Load` its `onto-state.yaml`: missing/invalid →
   finding; loaded but `!Archived` → finding. (Non-directory entries under
   `archive/` are ignored — the archive holds change directories.)

Each change/entry is keyed by its directory basename in the finding text so a
reader can locate it.

### D4 — Reuse `ValidateSkeleton` for phase-vs-artifacts

`ValidateSkeleton(changeDir)` already re-loads the state, derives the phase, and
checks `RequiredArtifacts(phase)` — precisely the "phase matches artifacts"
check. `doctor` calls it directly rather than re-implementing the artifact walk.
It re-loads the state internally; the small duplicate read is acceptable for a
diagnostic and keeps the check authoritative (single source of truth with
`onto status`).

## Risks / Trade-offs


```

Full source: openspec/changes/onto-doctor/design.md

## openspec/changes/onto-doctor/tasks.md

- Source: openspec/changes/onto-doctor/tasks.md
- Lines: 1-12
- SHA256: 5b5dd911685fea0e032a1837d449c29d3b4690274b0f94152ed80b94fb9ad5b3

```md
## 1. `onto doctor` command (`internal/ontocli`)

- [ ] 1.1 (TDD, RED first) `doctorCmd()` (`--dir` default ".") + `runDoctor(cmd, root)`: read-only, ungated. Accumulate findings, then: docs layout — stat each of `docsLayout`, a missing/non-dir path is a finding; active changes — `filepath.Glob(root/docs/changes/*/onto-state.yaml)`, per change: `ontostate.Load` (invalid → finding, skip rest of this change), `DerivePhase` (invalid → finding, skip), `ontostate.ValidateSkeleton(changeDir)` (err → finding), `ontostate.DepsResolved(root, st.Deps)` (non-empty → finding naming them), `st.Archived` (true → active-marked-archived finding); archive layout — `filepath.Glob(root/docs/changes/archive/*)`, per directory entry: `Load` its `onto-state.yaml` (missing/invalid → finding), else `!st.Archived` → finding. Findings print to `cmd.OutOrStdout()`; zero findings → print `healthy`, return nil; ≥1 → print `N problem(s) found` summary line and return a non-nil error. Each finding is keyed by directory basename.
- [ ] 1.2 (TDD, RED first) Tests via `NewRootCmd().SetArgs([]string{"doctor","--dir",tmp})` + `cmd.Execute()`, asserting the returned error is nil/non-nil AND stdout contains the expected finding text AND (negative cases) that no file was created/modified. Helper to seed a workspace root with `docs/{changes,specs,adr,guides}` and to seed active/archived changes at a given phase with chosen artifacts + deps. Cases: healthy (full layout, one valid phase-matching dep-resolved active change, one well-formed archive entry → nil error, stdout `healthy`); missing `docs/adr` → non-nil, names it; invalid active `onto-state.yaml` → non-nil, names change; phase-without-artifact (build phase, no plan.md) → non-nil, names missing artifact; unresolved dep → non-nil, names dep; active change with `archived: true` → non-nil; malformed archive entry (archive dir with state `archived:false` or missing state) → non-nil; ungated read-only (no homonto.toml, missing docs) → still runs, reports layout findings, writes nothing (assert no new files).
- [ ] 1.3 GREEN; register `doctorCmd()` on the root in `root.go`; `grep -nE "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go` empty (isolation held); gofmt/vet clean for `internal/ontocli`.
- [ ] 1.4 Commit: `feat(onto): 'onto doctor' reports workflow/project health (read-only, non-zero on findings)`

## 2. Regression and docs

- [ ] 2.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; E2E in a temp git workspace: build `onto`, run `onto doctor` on a healthy scaffolded workspace (exit 0, `healthy`), then break it (remove a docs dir / seed an invalid change) and confirm `onto doctor` exits non-zero naming the problem.
- [ ] 2.2 Update `docs/roadmap.md` "Immediate Next Work": onto #4 (`onto doctor`) landed — the onto binary now exposes advance/close/doctor/init/new/status/version; only dual-binary release packaging (#5) remains before `v0.1.0-rc.1`. No over-claim.
- [ ] 2.3 Commit all changes.

```

## openspec/changes/onto-doctor/specs/onto-binary/spec.md

- Source: openspec/changes/onto-doctor/specs/onto-binary/spec.md
- Lines: 1-83
- SHA256: 939f33fe28761c27f25f523d432fd28608d37d75a4cd1f5d0a68f79f6e382471

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: onto doctor reports workflow and project health

`onto doctor [--dir <root>]` SHALL be a strictly read-only, config-independent
diagnostic that reports the health of an onto workspace. It SHALL perform zero
writes, never construct a homonto config/engine, and never read `homonto.toml`.
It SHALL run regardless of whether the onto framework is installed (it is a
diagnostic, not a mutation, and is therefore NOT behind the framework-install
gate). `--dir` SHALL default to `.`.

`onto doctor` SHALL check, and surface each problem it finds as an individual
finding line:

- **docs layout**: `docs/changes`, `docs/specs`, `docs/adr`, and `docs/guides`
  each exist as directories under the root; a missing one is a finding.
- **active change state validity**: for each `docs/changes/*/onto-state.yaml`
  (the single `*` excludes archived changes, which live one level deeper), the
  state loads, validates, and derives a phase; a malformed or invalid file is a
  finding.
- **phase matches artifacts**: for each valid active change, every artifact
  required for its derived phase is present; a missing required artifact is a
  finding.
- **dependency and gate consistency**: for each valid active change, every
  dependency it lists is resolved (an archived `docs/changes/archive/*-<dep>`
  exists); an unresolved dependency is a finding. An active change whose state
  already records `archived: true` is a finding (an archived change belongs
  under `docs/changes/archive/`).
- **archive layout**: each `docs/changes/archive/*` entry is a directory holding
  a valid `onto-state.yaml` marked `archived: true`; a missing or invalid state
  file, or one not marked archived, is a finding.

On a healthy workspace `onto doctor` SHALL print a single `healthy` line and
exit 0. When one or more findings exist it SHALL print each finding and a count
summary and exit non-zero.

#### Scenario: healthy workspace reports healthy and exits 0

- **GIVEN** a root with the full `docs/{changes,specs,adr,guides}` layout, a valid active change whose artifacts match its phase and whose deps are resolved, and a well-formed archive entry
- **WHEN** `onto doctor` runs
- **THEN** it prints `healthy` and exits 0

#### Scenario: missing docs layout directory is a finding

- **GIVEN** a root missing `docs/adr`
- **WHEN** `onto doctor` runs
- **THEN** it reports the missing `docs/adr` directory and exits non-zero

#### Scenario: invalid onto-state.yaml is a finding

- **GIVEN** an active change whose `onto-state.yaml` is malformed or fails validation
- **WHEN** `onto doctor` runs
- **THEN** it reports the change as invalid, naming the problem, and exits non-zero

#### Scenario: phase not matching artifacts is a finding

- **GIVEN** an active change at a phase whose required artifacts are not all present (e.g. phase `build` without `plan.md`)
- **WHEN** `onto doctor` runs
- **THEN** it reports the missing required artifact and exits non-zero

#### Scenario: unresolved dependency is a finding

- **GIVEN** an active change whose `onto-state.yaml` lists a dependency that is not archived
- **WHEN** `onto doctor` runs
- **THEN** it reports the unresolved dependency and exits non-zero

#### Scenario: active change marked archived is a finding

- **GIVEN** an active change (under `docs/changes/<name>/`, not the archive) whose state records `archived: true`
- **WHEN** `onto doctor` runs
- **THEN** it reports the inconsistency and exits non-zero

#### Scenario: malformed archive entry is a finding

- **GIVEN** a `docs/changes/archive/<entry>` whose `onto-state.yaml` is missing, invalid, or not marked `archived: true`
- **WHEN** `onto doctor` runs
- **THEN** it reports the malformed archive entry and exits non-zero

#### Scenario: onto doctor is read-only and needs no framework install


```

Full source: openspec/changes/onto-doctor/specs/onto-binary/spec.md
