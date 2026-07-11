# Verification Report: onto-release-packaging (#5)

- **Change**: `onto-release-packaging` — dual-binary release packaging
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (scale: 7 tasks, 1 capability, 12 changed files)
- **Result**: PASS — no CRITICAL or IMPORTANT issues

## Scope (source diff vs base `5ccee3f`)

| File | Change |
|------|--------|
| `scripts/build-release.sh` | new — shared dual-binary packaging script (exec bit 100755) |
| `.github/workflows/release.yml` | inline homonto-only build+checksum → one script call |
| `.github/workflows/ci.yml` | add `onto` version-stamp smoke |
| `.gitignore` | ignore `/dist/` |
| `docs/roadmap.md`, `docs/release-notes.md` | onto #1–#5 complete; ships both binaries |

No Go source or dependency change.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks.md / plan tasks completed `[x]` | PASS (0 unchecked) |
| 2 | Implementation matches `design.md` decisions (D1 extract script, D2 separate archive per binary, D3 per-binary ldflag, D4 CI onto smoke) | PASS |
| 3 | Implementation matches Design Doc (`docs/superpowers/specs/2026-07-11-onto-release-packaging-design.md`) — script shape, release/CI diffs | PASS |
| 4 | All capability spec scenarios pass | PASS (see mapping) |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS (no Spec Patches) |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → evidence mapping

| Spec scenario | Evidence | Result |
|---|---|---|
| both binaries' archives for every target | packaging E2E: 12 archives (6 `homonto_*` + 6 `onto_*`) | PASS |
| each binary carries its own stamped version | extracted `homonto version`→`v0.0.0-test`, `onto version`→`v0.0.0-test` (own package ldflag) | PASS |
| windows archives are zips with .exe | `unzip -l` shows `homonto.exe`/`onto.exe` in `.zip` | PASS |
| CI smoke covers onto version stamp | `ci.yml` `onto version stamp smoke` step (build `./cmd/onto` + assert `ci-smoke`) | PASS |

## Commands run (verification evidence)

| Command | Result |
|---|---|
| `bash -n scripts/build-release.sh` | clean |
| `shellcheck scripts/build-release.sh` | clean |
| `bash scripts/build-release.sh v0.0.0-verify` | 12 archives, `sha256sum -c` → 12/12 OK |
| slash-in-version guard (`v1/beta`) | rejected, exit 1 (no partial release) |
| `go build ./...` (both binaries) | Success |
| `go test ./... -count=1` | 321 passed, 23 packages |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |
| workflows YAML parse (`release.yml`, `ci.yml`) | valid YAML |
| script exec bit | committed 100755 (bare invocation works in CI) |

## Local packaging E2E (temp `dist/`, cleaned after)

`bash scripts/build-release.sh v0.0.0-test` → `dist/` held exactly 12 archives
(6 homonto: 4 `.tar.gz` + 2 `.zip`; 6 onto: 4 `.tar.gz` + 2 `.zip`) + a
`SHA256SUMS` listing all 12; `sha256sum -c` verified all 12. Extracted the
linux/amd64 pair: the homonto archive held `homonto` + LICENSE + README, the onto
archive held `onto` + LICENSE + README; on this amd64 host both extracted
binaries reported `v0.0.0-test`. The windows amd64 archive was a `.zip`
containing `homonto.exe` + LICENSE + README.

## Code review (review_mode: standard)

One final lightweight review (correctness / security / edge cases): **no CRITICAL
or IMPORTANT findings.** Confirmed the `set -eu` AND-OR idiom is safe, both globs
always match (no unmatched-glob hazard), fail-loud on any target failure (no
silent under-count / unstamped binary), correct `cd dist` / SHA256SUMS scoping,
correct release.yml script invocation (exec bit + shebang; child sets its own
`set -eu`; non-zero propagates) with the publish glob and verify/publish steps
unchanged, and a correct ci.yml onto smoke (right build path + version package).
MINOR notes: a `/` in the version could nest archives and silently under-count —
**hardened** (the script now rejects `/` in the version, verified); and the
implicit `zip` dependency (present on `ubuntu-latest`) — informational.

## Conclusion

Verification PASS. `onto-release-packaging` completes #5 — the final onto binary
work item and the last release-gate packaging task. The dual-binary `homonto` +
`onto` product is complete; the remaining gate for `v0.1.0-rc.1` is the
maintainer's tag.
