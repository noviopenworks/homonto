---
comet_change: onto-release-packaging
role: technical-design
canonical_spec: openspec
---

# Dual-Binary Release Packaging — Technical Design

Deep refinement of `openspec/changes/onto-release-packaging/design.md`. The
open-phase design fixed the approach (extract a shared script, separate archive
per binary, per-binary version ldflag, CI onto smoke); this document nails the
script shape, the release/CI diffs, and the verification.

## `scripts/build-release.sh`

```bash
#!/usr/bin/env bash
# Cross-compile, package, and checksum BOTH binaries for every target.
# Usage: scripts/build-release.sh <version>
# Runnable locally (Go cross-compiles from any host); the release workflow
# calls it so the same code path runs on and off CI.
set -eu

VERSION="${1:?usage: build-release.sh <version>}"

TARGETS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"

# build_one <name> <buildpath> <versionpkg> <goos> <goarch>
build_one() {
  name="$1"; buildpath="$2"; versionpkg="$3"; goos="$4"; goarch="$5"
  bin="$name"
  [ "$goos" = "windows" ] && bin="$name.exe"
  out="${name}_${VERSION}_${goos}_${goarch}"
  echo "building $out"
  mkdir -p "dist/$out"
  cp LICENSE README.md "dist/$out/"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "-s -w -X ${versionpkg}.Version=${VERSION}" \
      -o "dist/$out/$bin" "$buildpath"
  if [ "$goos" = "windows" ]; then
    (cd dist && zip -qr "$out.zip" "$out")
  else
    tar -C dist -czf "dist/$out.tar.gz" "$out"
  fi
  rm -rf "dist/$out"
}

mkdir -p dist
for target in $TARGETS; do
  goos="${target%/*}"; goarch="${target#*/}"
  build_one homonto . github.com/noviopenworks/homonto/internal/cli    "$goos" "$goarch"
  build_one onto ./cmd/onto github.com/noviopenworks/homonto/internal/ontocli "$goos" "$goarch"
done

cd dist
# One SHA256SUMS over every archive (12), so downloaders can verify either binary.
sha256sum ./*.tar.gz ./*.zip > SHA256SUMS
cat SHA256SUMS
```

Notes:
- `${1:?…}` makes a missing version a hard error with usage — no silent empty
  stamp.
- `build_one` is the single build definition; homonto and onto differ only by
  the three leading args, so there is no copy-pasted loop body.
- `dist/` is created once; the function `rm -rf`s each staging dir after
  packaging, leaving only archives — safe to re-run (a second run overwrites
  archives of the same name).
- ldflags match today's homonto build exactly (`-s -w -trimpath CGO_ENABLED=0`),
  so binary characteristics are unchanged for homonto and identical for onto.

## `release.yml` diff

The "build release artifacts" step (inline homonto-only loop) and the
"checksums" step collapse into one:

```yaml
      - name: build release artifacts
        env:
          VERSION: ${{ github.ref_name }}
        run: scripts/build-release.sh "${VERSION}"
```

Everything else — checkout, setup-go, the `verify` guard (`gofmt -l`, `go vet`,
`go test`), prerelease detection, `--notes-file docs/release-notes.md`, and the
publish glob `dist/*.tar.gz dist/*.zip dist/SHA256SUMS` — is unchanged. The glob
now matches both binaries' archives because they all land in `dist/`.

## `ci.yml` diff

After the existing homonto "version stamp smoke", add:

```yaml
      - name: onto version stamp smoke
        run: |
          go build -ldflags "-X github.com/noviopenworks/homonto/internal/ontocli.Version=ci-smoke" -o /tmp/onto ./cmd/onto
          /tmp/onto version 2>&1 | tee /dev/stderr | grep -q "ci-smoke" || { echo "onto version not stamped"; exit 1; }
```

`onto version` prints `onto ci-smoke` via `cmd.Printf` (stdout); the `2>&1`
capture is defensive and mirrors the homonto smoke. `go build ./...` in the
existing "go build" step already compiles `./cmd/onto`, so no extra build gate.

## Verification (this is a CI/script change — verify by running the script)

1. **`bash -n scripts/build-release.sh`** and **`shellcheck scripts/build-release.sh`**
   (shellcheck present locally) — no syntax/lint errors.
2. **Local packaging E2E**: `bash scripts/build-release.sh v0.0.0-test` in a
   clean tree. Assert:
   - `dist/` holds exactly 12 archives — 6 `homonto_v0.0.0-test_*` (4 `.tar.gz`
     + 2 `.zip`) and 6 `onto_v0.0.0-test_*` (4 `.tar.gz` + 2 `.zip`) — plus
     `SHA256SUMS`.
   - `SHA256SUMS` lists all 12 and `(cd dist && sha256sum -c SHA256SUMS)` passes.
   - Extract `homonto_v0.0.0-test_linux_amd64.tar.gz` and
     `onto_v0.0.0-test_linux_amd64.tar.gz`: each contains the right binary +
     LICENSE + README; on an amd64 host the extracted binaries run and
     `homonto version` / `onto version` report `v0.0.0-test`.
   - A windows archive is a `.zip` and its binary is `homonto.exe` / `onto.exe`.
3. **Go suite unaffected**: `go build ./...` (both binaries), `go test ./...
   -count=1`, `go vet ./...`, `gofmt -l .` empty.
4. The `release.yml`/`ci.yml` YAML is well-formed (the workflow files parse;
   optionally `actionlint` if available — not required).

The only surface not locally reproducible is `gh release create`, which is
unchanged from the already-working homonto release path.

## Edge cases

- **Missing version arg** → `${1:?}` aborts with usage; CI always passes
  `github.ref_name`.
- **Re-run** → archives overwrite in place; `dist/` staging dirs are removed each
  iteration; `SHA256SUMS` regenerates. (A stale archive from a *different*
  version would linger; CI runs on a fresh checkout so this is a local-only
  caveat, noted for the E2E which uses a clean tree.)
- **`zip` absent** → windows packaging fails loudly (non-zero); present on the
  ubuntu runner and locally.

## Non-goals

No goreleaser, no signing/notarization, no new targets, no Go source change.
