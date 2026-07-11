---
change: onto-release-packaging
design-doc: docs/superpowers/specs/2026-07-11-onto-release-packaging-design.md
base-ref: 5ccee3f8b35cb9608fca1bc5d5bb16134330c9f4
---

# Plan: dual-binary release packaging (#5)

Ship both `homonto` and `onto` from the release pipeline. Extract packaging into
a shared, locally-runnable `scripts/build-release.sh`; separate archive per
binary (user-confirmed); shared `SHA256SUMS`. See the Design Doc for the exact
script, release/CI diffs, and verification. `tdd_mode: direct` — this is
shell/YAML infra with no unit-testable logic; verification is the local
packaging E2E.

## Task 1: packaging script + workflow rewiring

- [ ] 1.1 Add `scripts/build-release.sh` per the Design Doc: `#!/usr/bin/env
  bash`, `set -eu`, `VERSION="${1:?usage: build-release.sh <version>}"`,
  `build_one(name, buildpath, versionpkg, goos, goarch)` helper, loop the six
  targets building homonto (`.` / `internal/cli`) and onto (`./cmd/onto` /
  `internal/ontocli`) as separate archives (`.zip`+`.exe` for windows,
  `.tar.gz` otherwise), then `sha256sum ./*.tar.gz ./*.zip > SHA256SUMS`.
  `chmod +x`. Verify `bash -n` and `shellcheck` clean.
- [ ] 1.2 Rewire `.github/workflows/release.yml`: collapse the "build release
  artifacts" inline loop + "checksums" step into one step running
  `scripts/build-release.sh "${VERSION}"` (VERSION from `github.ref_name`).
  Leave verify/publish/prerelease/notes and the publish glob unchanged.
- [ ] 1.3 Extend `.github/workflows/ci.yml`: add an `onto` version-stamp smoke
  after the homonto one (build `./cmd/onto` with the `internal/ontocli.Version=
  ci-smoke` ldflag; assert `onto version` prints `ci-smoke`).
- [ ] 1.4 Commit: `build(release): ship both homonto and onto binaries (separate archives, shared SHA256SUMS)`

## Task 2: Verification and docs

- [ ] 2.1 Local packaging E2E (Design Doc "Verification"): `bash
  scripts/build-release.sh v0.0.0-test` in a clean tree → assert 12 archives (6
  homonto + 6 onto) + `SHA256SUMS`; `sha256sum -c SHA256SUMS` passes; extract
  the linux/amd64 pair → right binary + LICENSE + README each, and the extracted
  binaries report `v0.0.0-test`; a windows archive is a `.zip` with `.exe`.
  Confirm `go build ./...`, `go test ./... -count=1`, `go vet ./...`, `gofmt -l
  .` (empty) all still pass. Clean up the throwaway `dist/`.
- [ ] 2.2 Update `docs/roadmap.md` + `docs/release-notes.md`: #5 landed — release
  ships both binaries; onto binary work (#1–#5) COMPLETE; remaining for
  `v0.1.0-rc.1` is the maintainer tag + any remaining release-gate coverage. No
  over-claim (no tag cut).
- [ ] 2.3 Commit all changes.
