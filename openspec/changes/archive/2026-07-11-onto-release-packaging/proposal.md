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
