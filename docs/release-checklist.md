# Release Checklist

The repeatable steps for cutting a `homonto` release. This is the operational
companion to [`road-to-release.md`](road-to-release.md): the road-to-release file
is the gate for *whether* to release; this file is *how*.

Releases are driven by the `release` GitHub workflow
(`.github/workflows/release.yml`), which triggers on any pushed `v*` tag. It
re-runs the CI gates, cross-compiles every target, checksums the archives, and
publishes a GitHub release.

## Pre-tag verification

Run the full local gate from a clean worktree before tagging:

```sh
gofmt -l .            # expect no output
go mod tidy -diff     # expect no output
go vet ./...
go build ./...
go test ./...
go test -race ./...
./scripts/docker-test.sh
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Then confirm the repo dogfoods cleanly:

```sh
go run . status       # expect: No drift
go run . doctor       # expect all skills linked (a missing `pass` warning is fine)
```

## Tag and publish

1. Pick the version. Pre-releases use a suffix (`v0.1.0-rc.1`); a bare
   `vMAJOR.MINOR.PATCH` is a full release. The workflow marks any tag containing
   `-` as a GitHub pre-release automatically.
2. Tag an annotated tag on the commit that passed verification, and push it:

   ```sh
   git tag -a v0.1.0-rc.1 -m "v0.1.0-rc.1"
   git push origin v0.1.0-rc.1
   ```

3. The `release` workflow then:
   - re-runs gofmt/vet/test as a guard,
   - builds `linux`, `darwin`, and `windows` for `amd64` and `arm64`,
     stamping the tag into `homonto version` via `-ldflags`,
   - archives each target (`.tar.gz`, or `.zip` for Windows) with `LICENSE`
     and `README.md`,
   - writes a single `SHA256SUMS` over every archive,
   - creates the GitHub release with generated notes and all assets attached.

## Post-tag smoke install

From **outside** the repo, in a clean environment, verify the module installs
at the tag:

```sh
GOBIN=$(mktemp -d) go install github.com/noviopenworks/homonto@v0.1.0-rc.1
"$GOBIN"/homonto version    # expect the tagged version string
```

Then exercise the binary against a disposable home:

```sh
HOME=$(mktemp -d) # (or run in a container)
homonto init
# edit the generated homonto.toml minimally
homonto plan
homonto apply --yes
homonto status              # expect: No drift
homonto doctor
```

Verify a downloaded archive's checksum matches `SHA256SUMS`:

```sh
sha256sum -c SHA256SUMS --ignore-missing
```

## Rollback

A release is only ever additive to git history, so rollback is deletion plus a
follow-up, never a force-push:

1. Mark the bad GitHub release as a draft (or delete it) so it stops being
   offered:

   ```sh
   gh release delete v0.1.0-rc.1 --yes
   ```

2. Delete the tag locally and on the remote:

   ```sh
   git tag -d v0.1.0-rc.1
   git push origin :refs/tags/v0.1.0-rc.1
   ```

3. `go install ...@v0.1.0-rc.1` will keep working for anyone who already
   resolved it (the module proxy caches tags), so a broken release is corrected
   by shipping a higher patch/rc tag, not by expecting the old one to vanish.
   Never re-point an existing tag at a different commit.

## Security scanning decision (CodeQL / dependency-review)

For the v0.1.0 line, CodeQL and dependency-review are **deferred**, and this is
intentional rather than an oversight:

- `govulncheck` already runs in CI and scans both dependencies and the standard
  library for *called* known vulnerabilities — the highest-signal check for a
  small Go CLI with a tiny dependency set (`cobra`, `go-toml`, `sjson`/`gjson`).
- CodeQL's value grows with codebase size and untrusted input surfaces;
  `homonto` reads a local TOML the user owns and writes local files, so the
  marginal find rate over `go vet` + `govulncheck` is low for now.
- dependency-review gates *new* dependencies in PRs; with a near-static
  dependency list its overhead outweighs its signal today.

Revisit both when the dependency surface grows or the tool starts handling
untrusted remote input (e.g. fetching templates or remote config).
