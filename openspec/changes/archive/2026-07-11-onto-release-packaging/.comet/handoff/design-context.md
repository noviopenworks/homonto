# Comet Design Handoff

- Change: onto-release-packaging
- Phase: design
- Mode: compact
- Context hash: eb3d461f314fb1ba607a5c0eb30f80bb8ea7761fb41baa92d10deba57cf0ffbe

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-release-packaging/proposal.md

- Source: openspec/changes/onto-release-packaging/proposal.md
- Lines: 1-64
- SHA256: 9f49cb4f21d2425897457b0a026ef1e4d4f7cf3e155614647f71346677e6bc46

```md
## Why

The release gate for `v0.1.0-rc.1` is a dual-binary product, but the release
pipeline still ships only `homonto`. `.github/workflows/release.yml`
cross-compiles a single binary from `.` (the `homonto` main package) and never
builds `onto` (`./cmd/onto`), so a tagged release would omit half the product.
The dual-binary design's final open item is: "Release build packaging must be
updated to ship both `homonto` and `onto`." This change (#5, the last onto
binary work item) updates release + CI packaging to build, stamp, checksum, and
publish **both** binaries.

## What Changes

- Add `scripts/build-release.sh <version>`: a single, locally-runnable,
  `set -eu` build script that cross-compiles **both** binaries for all six
  targets (`{linux,darwin,windows} × {amd64,arm64}`) into `dist/`, then writes
  one `SHA256SUMS` over every archive. This extracts the packaging logic out of
  the workflow YAML so it is testable off CI and shared by both.
  - **Separate archive per binary**: each target yields two archives —
    `homonto_<version>_<os>_<arch>.{tar.gz,zip}` (containing `homonto` +
    LICENSE + README) and `onto_<version>_<os>_<arch>.{tar.gz,zip}` (containing
    `onto` + LICENSE + README) — 12 archives total, each independently
    downloadable and checksummed.
  - Each binary is version-stamped with its own ldflag: `homonto` via
    `-X …/internal/cli.Version=<version>`, `onto` via
    `-X …/internal/ontocli.Version=<version>`. Both use
    `CGO_ENABLED=0 -trimpath -ldflags "-s -w …"`, matching today's homonto build.
  - Windows binaries carry the `.exe` suffix and ship as `.zip`; others ship as
    `.tar.gz`. A single `SHA256SUMS` covers all 12 archives.
- Rewire `.github/workflows/release.yml` to call `scripts/build-release.sh
  "${VERSION}"` instead of its inline homonto-only loop, then publish
  `dist/*.tar.gz dist/*.zip dist/SHA256SUMS` (unchanged glob — now resolves to
  both binaries' archives). The verify guard step and prerelease/notes logic are
  unchanged.
- Extend `.github/workflows/ci.yml`: alongside the existing homonto
  version-stamp smoke, add an `onto` version-stamp smoke (`go build` `./cmd/onto`
  with the `internal/ontocli.Version` ldflag, assert `onto version` prints the
  stamp). `go build ./...` already covers compiling both binaries.
- Add a bundled release-notes acknowledgement (if needed) that the release now
  ships two binaries. No change to binary behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains a release-packaging guarantee — the release pipeline
  cross-compiles, version-stamps, checksums, and publishes both `homonto` and
  `onto` for all six OS/arch targets as separate per-binary archives under one
  shared `SHA256SUMS`.

## Impact

- New `scripts/build-release.sh` (the shared, testable packaging entrypoint).
- `.github/workflows/release.yml`: inline build loop → script call.
- `.github/workflows/ci.yml`: add `onto` version-stamp smoke.
- No Go source or dependency change; both binaries already build via
  `go build ./...`. `onto`'s version variable (`internal/ontocli.Version`) and
  `homonto`'s (`internal/cli.Version`) already exist and are ldflag-stampable.
- Completes #5 — the final onto binary work item and the last release-gate
  packaging task before `v0.1.0-rc.1`.

```

## openspec/changes/onto-release-packaging/design.md

- Source: openspec/changes/onto-release-packaging/design.md
- Lines: 1-93
- SHA256: 95ca644e695e30e5e2c6d29d71d156d45fd20b03c63d6c2fd6183dea91dda5e5

[TRUNCATED]

```md
## Context

`release.yml` today cross-compiles only `homonto` (built from `.`) via an inline
shell loop, packages one archive per target, checksums, and publishes on a `v*`
tag. `onto` (built from `./cmd/onto`) is never built. Both binaries already
carry ldflag-stampable `Version` variables (`internal/cli.Version`,
`internal/ontocli.Version`). #5 makes the release ship both, and does so through
a shared, testable script rather than more inline YAML.

## Goals / Non-Goals

**Goals**
- Ship both binaries for all six targets, each version-stamped, under one
  `SHA256SUMS`.
- Layout: **separate archive per binary** (12 archives) — the user-confirmed
  decision.
- Extract packaging into `scripts/build-release.sh` so it runs identically on
  and off CI and can be exercised locally.
- Keep the release workflow's guard/verify, prerelease detection, and notes
  logic unchanged.

**Non-Goals**
- No goreleaser / new tooling — stay with plain Go cross-compile + shell, the
  existing convention.
- No new targets, no changes to the six OS/arch pairs.
- No binary behavior change; no Go source change.
- No signing/notarization (out of scope for the first release).

## Decisions

### D1 — Extract `scripts/build-release.sh <version>`

A single `set -eu` script owns the packaging logic. It takes the version as
`$1`, cross-compiles both binaries for the six targets, and writes archives +
`SHA256SUMS` into `dist/`. `release.yml`'s "build release artifacts" +
"checksums" steps collapse to one `scripts/build-release.sh "${VERSION}"` call.

Rationale: packaging is the part most likely to silently break (a binary
omitted, a bad ldflag path, a checksum gap). Inline YAML can only be tested by
pushing a tag. A script can be run locally against every target (Go
cross-compiles from any host) and asserted on. It also removes duplication
between a would-be homonto loop and an onto loop.

### D2 — Separate archive per binary (user-confirmed)

Per target, two archives:
- `homonto_<version>_<os>_<arch>/` → `homonto[.exe]` + LICENSE + README;
- `onto_<version>_<os>_<arch>/` → `onto[.exe]` + LICENSE + README.

A small `build_one <name> <pkg> <version-pkg> <os> <arch>` shell function builds
and packages one binary so the two only differ by name/package/ldflag-path,
avoiding a copy-pasted second loop. `SHA256SUMS = sha256sum ./*.tar.gz ./*.zip`
covers all 12.

### D3 — Per-binary version ldflag

`homonto`: `-X github.com/noviopenworks/homonto/internal/cli.Version=<v>`
(built from `.`). `onto`:
`-X github.com/noviopenworks/homonto/internal/ontocli.Version=<v>` (built from
`./cmd/onto`). Both keep `CGO_ENABLED=0 -trimpath -ldflags "-s -w …"` exactly as
homonto builds today, so the only new axis is the second binary.

### D4 — CI onto version-stamp smoke

`ci.yml` gains a step mirroring the homonto version smoke: build `./cmd/onto`
with the `internal/ontocli.Version=ci-smoke` ldflag and assert `onto version`
prints `ci-smoke`. `onto` prints via `cmd.Printf` (stdout) — unlike homonto's
`version` which the existing smoke captures with `2>&1`; the onto smoke uses the
same `2>&1` capture defensively. `go build ./...` already compiles both, so no
separate build gate is needed.

## Risks / Trade-offs

- **Cannot fully run the GitHub Actions env locally.** Mitigation: the packaging
  logic lives in a plain script that IS runnable locally; verification builds all
  12 archives on this host, extracts them, and asserts each contains the right
  stamped binary + LICENSE + README, and that SHA256SUMS lists all 12 and
  verifies. The only unverified surface is the thin `gh release create` glue,
  which is unchanged from the working homonto release.
- **Archive count doubles (6 → 12).** Accepted — it is the user-confirmed layout

```

Full source: openspec/changes/onto-release-packaging/design.md

## openspec/changes/onto-release-packaging/tasks.md

- Source: openspec/changes/onto-release-packaging/tasks.md
- Lines: 1-46
- SHA256: 784a8a169781afb537562a45453c60f132f17aa5659083cfb13f2f5cb0b7ccc1

```md
## 1. `scripts/build-release.sh` — shared dual-binary packaging

- [ ] 1.1 Add `scripts/build-release.sh` (`#!/usr/bin/env bash`, `set -eu`),
  taking `<version>` as `$1` (error + usage if missing). Define a `build_one`
  function `(name, buildpath, versionpkg, goos, goarch, version)` that: sets
  `bin=<name>` (`.exe` for windows), `out=<name>_<version>_<goos>_<goarch>`,
  `mkdir -p dist/$out`, `cp LICENSE README.md dist/$out/`, builds with
  `CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags "-s -w -X
  <versionpkg>.Version=<version>" -o dist/$out/$bin <buildpath>`, then packages
  (`zip -qr` for windows, `tar -C dist -czf` otherwise) and `rm -rf dist/$out`.
  Loop over the six targets (`linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
  windows/amd64 windows/arm64`) calling `build_one` twice per target: `homonto`
  from `.` with `internal/cli`, and `onto` from `./cmd/onto` with
  `internal/ontocli`. Then `cd dist && sha256sum ./*.tar.gz ./*.zip > SHA256SUMS`.
  `mkdir -p dist` up front; make idempotent (safe to re-run). `chmod +x`.
- [ ] 1.2 Rewire `.github/workflows/release.yml`: replace the "build release
  artifacts" inline loop AND the "checksums" step with a single step running
  `scripts/build-release.sh "${VERSION}"` (VERSION from `github.ref_name`).
  Leave the verify step, publish step, prerelease detection, and
  `--notes-file docs/release-notes.md` unchanged; the publish glob stays
  `dist/*.tar.gz dist/*.zip dist/SHA256SUMS` (now resolves to both binaries).
- [ ] 1.3 Extend `.github/workflows/ci.yml`: after the existing homonto
  "version stamp smoke", add an `onto` version-stamp smoke — `go build -ldflags
  "-X github.com/noviopenworks/homonto/internal/ontocli.Version=ci-smoke" -o
  /tmp/onto ./cmd/onto` then `/tmp/onto version 2>&1 | grep -q ci-smoke` (fail
  otherwise).
- [ ] 1.4 Commit: `build(release): ship both homonto and onto binaries (separate archives, shared SHA256SUMS)`

## 2. Verification and docs

- [ ] 2.1 Local packaging E2E: run `bash scripts/build-release.sh v0.0.0-test`
  in a clean checkout (requires `zip`). Assert `dist/` has 12 archives (6
  `homonto_*` + 6 `onto_*`) + `SHA256SUMS`; `SHA256SUMS` lists all 12 and
  `sha256sum -c SHA256SUMS` passes; extract the `linux/amd64` pair and confirm
  each archive holds the right binary + LICENSE + README, and that the extracted
  `homonto`/`onto` (built for host arch if amd64) report the stamped version.
  Lint the script: `bash -n scripts/build-release.sh` (and `shellcheck` if
  available). Confirm `go build ./...` still builds both binaries; keep the full
  Go suite green (`go test ./... -count=1`, `go vet`, `gofmt -l .`).
- [ ] 2.2 Update `docs/roadmap.md`: #5 dual-binary release packaging landed — the
  release pipeline now ships both binaries; the onto binary work (#1–#5) is
  COMPLETE. Update the "Immediate Next Work" framing (onto binary done; what
  remains for `v0.1.0-rc.1` is the maintainer tag + any remaining release-gate
  coverage). Update `docs/release-notes.md` if it enumerates shipped binaries.
  No over-claim (do not claim a tag was cut).
- [ ] 2.3 Commit all changes.

```

## openspec/changes/onto-release-packaging/specs/onto-binary/spec.md

- Source: openspec/changes/onto-release-packaging/specs/onto-binary/spec.md
- Lines: 1-47
- SHA256: cc90484081fdf21bae1639e67191042c71a96bd04e0a06fd708ded6856e4b732

```md
## ADDED Requirements

### Requirement: Release packaging ships both binaries

The release pipeline SHALL cross-compile, version-stamp, checksum, and publish
**both** the `homonto` and `onto` binaries for every supported target. A shared,
locally-runnable build script `scripts/build-release.sh <version>` SHALL be the
single source of the packaging logic, invoked by the release workflow so the
same code path runs on and off CI.

For each of the six targets (`linux/amd64`, `linux/arm64`, `darwin/amd64`,
`darwin/arm64`, `windows/amd64`, `windows/arm64`) the script SHALL produce a
**separate archive per binary**:

- `homonto_<version>_<os>_<arch>` containing the `homonto` binary plus `LICENSE`
  and `README.md`;
- `onto_<version>_<os>_<arch>` containing the `onto` binary plus `LICENSE` and
  `README.md`.

Windows archives SHALL be `.zip` and carry the `.exe` suffix on the binary;
other targets SHALL be `.tar.gz`. Each binary SHALL be built with
`CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w -X <pkg>.Version=<version>"`
where `<pkg>` is `github.com/noviopenworks/homonto/internal/cli` for `homonto`
and `github.com/noviopenworks/homonto/internal/ontocli` for `onto`. A single
`SHA256SUMS` file SHALL cover every produced archive (12 in total).

#### Scenario: release build produces both binaries' archives for every target

- **GIVEN** the repository at a clean checkout and a version string
- **WHEN** `scripts/build-release.sh <version>` runs
- **THEN** `dist/` contains a `homonto_<version>_<os>_<arch>` archive and an `onto_<version>_<os>_<arch>` archive for each of the six targets (12 archives), and a `SHA256SUMS` listing all of them

#### Scenario: each binary carries its own stamped version

- **WHEN** the release build stamps the binaries
- **THEN** the `homonto` binary reports `<version>` via `homonto version` and the `onto` binary reports `<version>` via `onto version`, each stamped through its own package's `Version` ldflag

#### Scenario: windows archives are zips with .exe binaries

- **WHEN** the release build targets `windows/amd64` or `windows/arm64`
- **THEN** the produced archives are `.zip` files and the binary inside is named `homonto.exe` / `onto.exe`

#### Scenario: CI smoke covers the onto version stamp

- **GIVEN** the CI workflow
- **WHEN** it runs the version-stamp smoke checks
- **THEN** it stamps and runs `onto version` (in addition to `homonto version`) and fails if the stamped version is not reported

```
