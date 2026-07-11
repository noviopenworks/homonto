# Brainstorm Summary

- Change: onto-release-packaging
- Date: 2026-07-11

## Confirmed Technical Approach

Extract packaging into `scripts/build-release.sh <version>` (`set -eu`) with a
`build_one(name, buildpath, versionpkg, goos, goarch, version)` helper. Loop the
six targets, calling it twice per target (homonto from `.` / `internal/cli`;
onto from `./cmd/onto` / `internal/ontocli`), producing **separate archives per
binary** (user-confirmed: 12 archives), then `sha256sum ./*.tar.gz ./*.zip >
SHA256SUMS`. `release.yml` collapses its inline build+checksum steps to one
script call; `ci.yml` gains an `onto` version-stamp smoke.

## Key Trade-offs and Risks

- GitHub Actions env can't be fully run locally → mitigate by putting all
  packaging logic in a locally-runnable script and E2E-building all 12 archives
  on this host, asserting contents + SHA256SUMS verify. Only the unchanged `gh
  release create` glue stays unverified.
- Archive count doubles 6→12: accepted (user-confirmed separate-archive layout).
- Windows `.zip` needs `zip` CLI: present on the ubuntu runner and locally.

## Testing Strategy

`bash scripts/build-release.sh v0.0.0-test` in a clean tree → assert 12 archives
+ SHA256SUMS, `sha256sum -c` passes, extracted linux/amd64 pair holds the right
stamped binary + LICENSE + README. `bash -n` + `shellcheck` the script. Keep the
Go suite green; `go build ./...` builds both binaries.

## Spec Patches

None. The delta spec already carries the release-packaging requirement + four
scenarios (both-archives-per-target, per-binary version stamp, windows zips,
CI onto smoke).
