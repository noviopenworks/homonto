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
