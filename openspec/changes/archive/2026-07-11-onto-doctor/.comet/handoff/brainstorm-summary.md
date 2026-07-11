# Brainstorm Summary

- Change: onto-doctor
- Date: 2026-07-11

## Confirmed Technical Approach

Add `onto doctor [--dir]` as a read-only, ungated, config-independent diagnostic
in `internal/ontocli/doctor.go`. It assembles existing `ontostate` primitives
(`Load`, `Validate`, `DerivePhase`, `ValidateSkeleton`, `RequiredArtifacts`,
`DepsResolved`) and `ontocli.docsLayout` into a single health report. No new
`ontostate` exported API is introduced.

Check order (deterministic output):
1. docs layout — each of `docs/{changes,specs,adr,guides}` exists as a dir.
2. active changes — glob `docs/changes/*/onto-state.yaml` (the single `*`
   excludes the archive): per change Load→Validate→DerivePhase→ValidateSkeleton
   →DepsResolved→Archived-flag consistency.
3. archive layout — glob `docs/changes/archive/*`: each directory entry must
   hold a valid `onto-state.yaml` with `archived: true`.

Findings accumulate into a slice; each prints to stdout keyed by directory
basename. Zero findings → print `healthy`, return nil (exit 0). ≥1 finding →
print `N problem(s) found` and return a non-nil error → `main` prints
`error: …` to stderr and exits 1. Root already sets SilenceErrors/SilenceUsage,
so no cobra usage dump.

## Key Trade-offs and Risks

- Double-load of state (Validate path + inside ValidateSkeleton): accepted —
  diagnostic is not hot-path; reusing ValidateSkeleton keeps the phase-vs-
  artifacts check identical to `onto status`'s single source of truth.
- Per-change short-circuit only where a later check depends on an earlier one
  (cannot derive phase / validate skeleton / read deps from an unloadable
  state) to avoid nonsensical cascade findings.
- Exit-code coupling to `main` translating the returned error into a non-zero
  exit — same mechanism every other onto command uses; no new coupling.

## Testing Strategy

TDD, RED first. Table/case tests via `NewRootCmd().SetArgs(["doctor","--dir",
tmp])` + `Execute()`, asserting (a) returned error nil/non-nil, (b) stdout
contains the expected finding text, (c) negative cases write nothing (assert no
new files). Seed helpers build a workspace root with the docs layout and seed
active/archived changes at chosen phases/artifacts/deps. Cases: healthy;
missing docs dir; invalid active state; phase-without-artifact; unresolved dep;
active marked archived; malformed archive entry; ungated read-only with no
homonto.toml. Plus full regression (both binaries build, race, vet, gofmt, mod
tidy) and an E2E healthy→broken run.

## Spec Patches

None. The delta spec (`specs/onto-binary/spec.md`) already carries the
requirement plus eight acceptance scenarios covering every check and the
read-only/ungated invariant; no supplementary scenarios needed.
