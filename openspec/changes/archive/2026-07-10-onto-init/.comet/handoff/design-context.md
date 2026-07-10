# Comet Design Handoff

- Change: onto-init
- Phase: design
- Mode: full
- Context hash: 9311bd7928ae6599755dcfeb1db416a17fc75a6f43b30d728f81231bf4b5a4f9

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-init/proposal.md

- Source: openspec/changes/onto-init/proposal.md
- Lines: 1-53
- SHA256: 9c910ce48bf5bedb3630a5dc5ca3aaa2b755e25a106a8086ef848b640004be9e

```md
## Why

Change #1 (`onto-binary-foundation`, archived) shipped the `onto` binary with the
`onto-state.yaml` model and read-only `onto status`. This change (#2 of the
five-change `onto` decomposition) adds `onto init` — the first MUTATING command —
which scaffolds the `docs/` workflow layout. Per the dual-binary design, `onto`
is managed by Homonto: `onto init` may create the workspace only after the
project declares and applies `[frameworks.onto]` through Homonto; if that
framework install is missing, it directs the user to initialize/apply Homonto
first. This keeps `onto` a Homonto-managed operator, not an alternate installer.

## What Changes

- Add `onto init`: scaffolds the onto workflow layout
  `docs/{changes,specs,adr,guides}/` under the workspace root, idempotently
  (existing files/dirs are preserved, never clobbered), and reports what it
  created vs. skipped. It does NOT create `homonto.toml` (that is `homonto init`).
- **Framework-install gate** (mutating precondition): `onto init` requires (a) a
  `homonto.toml` at the workspace root declaring `[frameworks.onto]`, and (b) the
  onto framework materialized/installed by Homonto (evidence:
  `.homonto/catalog/skills/onto/` exists). If `homonto.toml` is absent or lacks
  `[frameworks.onto]`, or the framework is declared but not yet applied, `onto
  init` prints a clear message telling the user to run `homonto init` / declare
  `[frameworks.onto]` / run `homonto apply`, and exits non-zero WITHOUT creating
  any `docs/` files.
- `onto init` is additive to the `onto-binary` capability; `onto status`
  (read-only, config-independent) and the state model are unchanged.
- This change does NOT add phase-gate enforcement / skeleton create-validate
  (#3), `onto doctor` (#4), or release packaging (#5).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains the `onto init` command and the framework-install gate
  precondition for mutating operations. The foundation's read-only `onto status`
  and state model are unchanged.

## Impact

- New `internal/ontocli/init.go` (`initCmd()`), registered on the onto root.
- New gate helper — reads `homonto.toml` for `[frameworks.onto]` and checks the
  materialized onto framework. Prefer reusing `internal/config.Load` to parse the
  config and read `Config.Frameworks["onto"]`; check `.homonto/catalog/skills/onto`
  for the applied evidence. `onto init` must NOT run the projection engine.
- New `internal/ontocli/init_test.go`.
- No change to `homonto`, `internal/cli`, adapters, engine, config, catalog, or
  `internal/ontostate`.
- Advances the `onto` binary toward the dual-binary release gate (#2 of 5).

```

## openspec/changes/onto-init/design.md

- Source: openspec/changes/onto-init/design.md
- Lines: 1-71
- SHA256: cbaeb056697c8338d8bacc36eccd2be3a27a29f4160efeea02a942b4b239b3f4

```md
## Context

`onto-binary-foundation` (#1, archived) shipped the `onto` binary,
`internal/ontostate`, and read-only `onto status`. `onto init` (#2) is the first
mutating command: it scaffolds `docs/{changes,specs,adr,guides}` but only after
the project has declared and applied `[frameworks.onto]` through Homonto (the
dual-binary design's "managed, not an alternate installer" rule).

## Goals / Non-Goals

**Goals:** `onto init` scaffolds the four docs dirs idempotently; a
framework-install gate refuses (non-zero, no writes) when homonto.toml is absent,
lacks `[frameworks.onto]`, or the framework is unapplied; clear guidance messages.

**Non-Goals:** phase-gate enforcement / skeleton create-validate (#3), `onto
doctor` (#4), release packaging (#5); creating `homonto.toml` (that is `homonto
init`); running the projection engine; changing `onto status` or the state model.

## Decisions

**D1 — `internal/ontocli/init.go` with an explicit gate then scaffold.** `onto
init` runs the gate first; only if it passes does it scaffold. Registered on the
onto root alongside `version`/`status`.

**D2 — Gate via a lightweight homonto.toml read + a filesystem applied-check,
NOT the full config engine.** To keep `onto` decoupled from homonto's projection
pipeline (per #1's isolation) and avoid failing on unrelated config-validation
errors, the gate reads `homonto.toml` directly with `go-toml/v2` and checks only
for a `[frameworks.onto]` table (a `[frameworks]` map containing key `onto`). It
does NOT call `internal/config.Load` (which pulls the catalog and validates
models/all resources) and does NOT construct the engine/adapters. Applied
evidence is a filesystem check: `.homonto/catalog/skills/onto/` exists next to
`homonto.toml` (the materialized onto framework). Rationale: minimal coupling,
fast, and robust to unrelated config problems; the check is intentionally
structural, not a full config validation. Alternative (`config.Load`) rejected —
too much coupling and it would reject a config with any unrelated validation
error, blocking a legitimate `onto init`.

Gate order (first failure wins, each exits non-zero with specific guidance, no
writes):
1. `homonto.toml` missing → "run `homonto init`".
2. `homonto.toml` present but no `[frameworks.onto]` → "declare `[frameworks.onto]` and run `homonto apply`".
3. `[frameworks.onto]` present but `.homonto/catalog/skills/onto/` missing → "run `homonto apply`".

**D3 — Idempotent scaffold, skip-existing, report created vs skipped.** Mirror
`internal/scaffold.Init`'s skip-existing behavior: for each of
`docs/{changes,specs,adr,guides}`, `os.MkdirAll` (idempotent) and track whether
it pre-existed to report created vs skipped. Directory-only scaffold for this
increment (no template files inside — skeleton content is #3's concern). Never
overwrite. A `--dir` flag (default `.`) selects the workspace root, consistent
with `onto status`.

## Risks / Trade-offs

- **Applied-evidence heuristic** (`.homonto/catalog/skills/onto/`) → it is the
  materialization path homonto uses; if that layout changes, the check updates in
  one place. Acceptable structural proxy for "framework applied".
- **Lightweight TOML read vs config.Load** → the gate does not fully validate the
  config; it only needs `[frameworks.onto]` presence. This is deliberate (D2).
- **Directory-only scaffold** → no README/templates yet; #3 adds skeleton
  content. Keeps this change minimal.

## Migration Plan

Additive: new `internal/ontocli/init.go` + test; register one subcommand. No
change to existing packages/binaries. Rollback is removing the command.

## Open Questions

None blocking. Skeleton file templates inside the scaffolded dirs are deferred to
#3 (phase-gates / skeleton create-validate).

```

## openspec/changes/onto-init/tasks.md

- Source: openspec/changes/onto-init/tasks.md
- Lines: 1-17
- SHA256: 1c0f0983da119f38dc1efbe997056b94316902deef71b4ae7d5ed75c4c047a12

```md
## 1. Framework-install gate

- [ ] 1.1 Add a gate helper in `internal/ontocli/init.go` (or a small sibling): given a workspace root, return a typed result / error for: (a) `homonto.toml` missing, (b) present but no `[frameworks.onto]` table, (c) `[frameworks.onto]` present but `.homonto/catalog/skills/onto/` missing, (d) OK. Read homonto.toml with `go-toml/v2` (a minimal struct with `Frameworks map[string]any` — presence of key `onto` is enough); do NOT call `internal/config.Load` and do NOT construct the engine
- [ ] 1.2 Unit tests (TDD, RED first) for all four gate outcomes over temp workspaces, asserting the guidance message content and that NO `docs/` files are created in the three failing cases

## 2. `onto init` command + scaffold

- [ ] 2.1 Implement `initCmd()` with a `--dir` flag (default "."): run the gate; on failure print the specific guidance and return a non-zero error WITHOUT touching `docs/`; on success scaffold `docs/{changes,specs,adr,guides}` via `os.MkdirAll`, tracking created-vs-preexisting, and print the report; never overwrite existing paths
- [ ] 2.2 Register `initCmd()` on the onto root in `internal/ontocli/root.go`'s `NewRootCmd()`
- [ ] 2.3 Tests (TDD, RED first): in a prepared workspace (homonto.toml with `[frameworks.onto]` + a fake `.homonto/catalog/skills/onto/` dir) init creates the four dirs and reports created, exit 0; a second run is idempotent (pre-existing dirs + any user file under docs/ untouched, reported skipped); gate-failure cases create no docs/ files and exit non-zero
- [ ] 2.4 Confirm `onto init` does not import/run `internal/engine` or `internal/adapter`; keep the ontocli isolation from #1 (no homonto projection engine)

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; confirm both binaries build and `onto init --help` shows the command
- [ ] 3.2 Update `docs/roadmap.md` "Immediate Next Work": mark onto #2 (`onto init`) landed; remaining onto work = phase-gates (#3), doctor (#4), dual-binary packaging (#5). Do not over-claim
- [ ] 3.3 Commit all changes

```

## openspec/changes/onto-init/specs/onto-binary/spec.md

- Source: openspec/changes/onto-init/specs/onto-binary/spec.md
- Lines: 1-58
- SHA256: 7cf67bc9f111a8371a3718126560e5e45971fa19844d162a6a014016dae3295b

```md
## ADDED Requirements

### Requirement: onto init scaffolds the workflow layout

`onto init` SHALL scaffold the onto workflow directory layout under the
workspace root: `docs/changes/`, `docs/specs/`, `docs/adr/`, and `docs/guides/`.
It SHALL be idempotent — an existing directory or file is preserved and never
overwritten — and it SHALL report which paths it created versus skipped. `onto
init` SHALL NOT create `homonto.toml` (that is `homonto init`'s job) and SHALL
NOT run the Homonto projection engine.

#### Scenario: init creates the docs layout in a prepared workspace

- **GIVEN** a workspace whose `homonto.toml` declares `[frameworks.onto]` and whose onto framework has been applied by Homonto
- **WHEN** `onto init` runs
- **THEN** `docs/changes/`, `docs/specs/`, `docs/adr/`, and `docs/guides/` exist and the command reports the created paths, exiting 0

#### Scenario: init is idempotent

- **GIVEN** a workspace where `onto init` already created the layout (and a user has added content under `docs/`)
- **WHEN** `onto init` runs again
- **THEN** existing directories and files are left untouched, newly-created paths (if any) are reported as created and pre-existing ones as skipped, and the command exits 0

### Requirement: onto init requires the Homonto-managed framework install

`onto init` is a mutating command and SHALL require that the project has declared
and applied `onto` through Homonto before it creates any `docs/` files:

- If `homonto.toml` is absent at the workspace root, `onto init` SHALL print a
  message directing the user to run `homonto init`, and exit non-zero.
- If `homonto.toml` exists but does not declare `[frameworks.onto]`, `onto init`
  SHALL print a message directing the user to declare `[frameworks.onto]` and run
  `homonto apply`, and exit non-zero.
- If `[frameworks.onto]` is declared but the onto framework has not been applied
  (no materialized evidence such as `.homonto/catalog/skills/onto/`), `onto init`
  SHALL print a message directing the user to run `homonto apply`, and exit
  non-zero.

In every failing case `onto init` SHALL NOT create, modify, or delete any file
under `docs/`.

#### Scenario: init refuses without homonto.toml

- **GIVEN** a workspace with no `homonto.toml`
- **WHEN** `onto init` runs
- **THEN** it prints guidance to run `homonto init`, creates no `docs/` files, and exits non-zero

#### Scenario: init refuses when frameworks.onto is not declared

- **GIVEN** a `homonto.toml` that does not declare `[frameworks.onto]`
- **WHEN** `onto init` runs
- **THEN** it prints guidance to declare `[frameworks.onto]` and run `homonto apply`, creates no `docs/` files, and exits non-zero

#### Scenario: init refuses when the framework is declared but not applied

- **GIVEN** a `homonto.toml` declaring `[frameworks.onto]` but no applied evidence (no `.homonto/catalog/skills/onto/`)
- **WHEN** `onto init` runs
- **THEN** it prints guidance to run `homonto apply`, creates no `docs/` files, and exits non-zero

```
