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
