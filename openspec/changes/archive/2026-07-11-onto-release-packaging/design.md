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
  and keeps each binary independently downloadable/verifiable.
- **Windows zip on a Linux host** needs the `zip` CLI. CI's ubuntu runner has it;
  local verification requires `zip` present (documented in the plan's E2E).

## Migration Plan

Additive to the release/CI pipeline; no consumer or code migration. The first
tag built after this lands ships both binaries.

## Open Questions

None. The one user-facing decision (bundle vs separate archives) is resolved:
separate archive per binary.
