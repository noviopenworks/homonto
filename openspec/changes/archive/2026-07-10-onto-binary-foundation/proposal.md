## Why

The v0.1.0-rc.1 release gate is dual-binary: `homonto` (the deterministic
config projector) plus `onto` (the managed spec-driven workflow operator). Today
only `homonto` exists in source (`main.go` at the repo root); there is no `onto`
binary. `onto` is the last release blocker
(`docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`). The full
`onto` binary is large, so it is split into five changes (see below); this
change delivers the **foundation increment** everything else builds on: the
second binary exists and can inspect an existing `onto` workspace read-only.

### Onto binary decomposition (this change is #1)

1. **onto-binary-foundation** (this change) — second `package main`, Cobra root,
   the `onto-state.yaml` model (parse/write/validate/derive phase), and
   `onto status` (read-only, works without config).
2. onto-init — `onto init` scaffolds `docs/{changes,specs,adr,guides}`, gated on
   `[frameworks.onto]` being applied through Homonto. (depends on #1)
3. onto-phase-gates — skeleton create/validate, structural-invariant enforcement,
   valid-gate-only transitions, dependency resolution, archive/close and
   dirty-worktree rules. (depends on #1, #2)
4. onto-doctor — `onto doctor` workflow health. (depends on #1, #3)
5. dual-binary-release-packaging — release CI builds and publishes both binaries
   with a shared `SHA256SUMS`; install docs. (depends on #1)

## What Changes

- Add a second `package main` at `cmd/onto/` that builds the `onto` binary via
  `go build ./cmd/onto` (and is installable with `go install .../cmd/onto`),
  leaving the existing root `homonto` binary untouched.
- Add a Cobra root command `onto` mirroring `homonto`'s CLI construction style
  (`internal/cli`), stamped with a `version` the same way `homonto version` is.
- Add an `onto-state.yaml` model in a new `internal/ontostate` package:
  parse/serialize the change state file, validate its structure, and derive the
  current phase from its fields. The file is named **`onto-state.yaml`** (no
  migration/back-compat for the legacy `state.yaml` name — pre-release).
- Add `onto status`: a **read-only degraded** command that inspects an existing
  `docs/` workspace and its `onto-state.yaml` files WITHOUT requiring
  `homonto.toml` or `[frameworks.onto]`, for diagnostics and recovery. It reports
  each change's derived phase and flags missing/invalid state; it never writes.
- This change does NOT add mutating commands (`init`, phase transitions,
  `doctor`), the docs scaffolder, or release packaging — those are changes #2–#5.

## Capabilities

### New Capabilities

- `onto-binary`: the managed `onto` workflow-operator binary. This change
  introduces the capability's foundation — the binary and CLI root, the
  `onto-state.yaml` state model (parse/validate/derive-phase), and the read-only
  `onto status` command. Later changes (#2–#4) extend this same capability with
  `init`, phase-gate enforcement, and `doctor`.

### Modified Capabilities

None. This is a net-new binary and capability; it does not change `homonto`'s
config-projection behavior or any existing spec.

## Impact

- New `cmd/onto/main.go` (second `package main`); the root `homonto` binary and
  `main.go` are unchanged.
- New `internal/ontostate/` (state model: types, parse, serialize, validate,
  derive-phase) with tests.
- New `internal/cli` entry for the `onto` root + `status` subcommand (either a
  new file in `internal/cli` or a sibling package; follow the existing
  `NewRootCmd` pattern).
- No change to `internal/adapter`, `internal/engine`, `internal/config`, or the
  catalog — `onto` is independent of the projection pipeline in this increment.
- Release packaging still ships only `homonto` after this change; dual-binary
  packaging is change #5.
- Advances the roadmap's sole remaining Immediate Next Work item (the `onto`
  binary) by landing its foundation.
