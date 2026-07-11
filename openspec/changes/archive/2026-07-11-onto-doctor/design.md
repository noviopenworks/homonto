## Context

`onto doctor` is the last onto workflow command (#4). It is a diagnostic peer to
`homonto doctor`: where `homonto doctor` checks installation/projection health,
`onto doctor` checks *workflow and project* health. All the primitives it needs
already exist in `internal/ontostate` (`Load`, `Validate`, `DerivePhase`,
`ValidateSkeleton`, `RequiredArtifacts`, `DepsResolved`) and in
`internal/ontocli` (`docsLayout`). This change is therefore an assembly of
existing read-only primitives into one report — no new `ontostate` API.

## Goals / Non-Goals

**Goals**
- One command that answers "is this onto workspace healthy?" with a clear list
  of findings and a CI-gateable exit code.
- Strictly read-only and config-independent, exactly like `onto status`.
- Reuse existing `ontostate`/`ontocli` primitives; add no exported helper.

**Non-Goals**
- Fixing problems (`doctor` diagnoses; it never mutates). No `--fix`.
- Installation/projection health — that is `homonto doctor`'s job.
- Configurable artifact roots outside `docs/` (a first-release non-goal).
- Checking `onto-state.yaml` content beyond what `Validate`/`DerivePhase`/
  `ValidateSkeleton`/`DepsResolved` already enforce.

## Decisions

### D1 — Ungated, read-only, `--dir` default `.`

`onto doctor` is modeled on `onto status`, not `onto init`: it is a read-only
diagnostic, so it is NOT behind the framework-install `gate`. A workspace with
no `homonto.toml` / no installed framework is a valid thing to diagnose — the
missing `docs/` layout is reported as a finding rather than causing a refusal.
It performs zero writes and imports none of homonto's projection packages
(`internal/{cli,engine,config,adapter,catalog}`), preserving onto's isolation.

### D2 — Findings model + exit code

`runDoctor` accumulates finding strings into a slice, printing each to stdout as
it goes (or collecting then printing — implementation detail). At the end:
- zero findings → print `healthy`, return `nil` (exit 0);
- ≥1 finding → print a count summary (e.g. `N problem(s) found`) and return a
  non-nil error so the process exits non-zero.

The root command sets `SilenceErrors`/`SilenceUsage`, and `cmd/onto/main.go`
prints `error: <err>` to stderr and exits 1, so returning a summary error gives
a clean non-zero exit without a cobra usage dump. Findings themselves print to
stdout; only the one-line summary rides the error to stderr.

### D3 — Check set and ordering

Checks run in a fixed, stable order so output is deterministic:
1. **docs layout** — stat each of `docsLayout`; a non-directory / missing path
   is a finding.
2. **active changes** — `filepath.Glob(root/docs/changes/*/onto-state.yaml)`
   (the single `*` cannot cross a separator, so it structurally excludes
   `docs/changes/archive/<name>/…`, exactly as `onto status` relies on). For
   each: `Load`→invalid finding; else `DerivePhase`→invalid finding; else
   `ValidateSkeleton`→missing-artifact finding; `DepsResolved(root, st.Deps)`
   →unresolved-dep finding; `st.Archived == true`→active-marked-archived
   finding.
3. **archive layout** — `filepath.Glob(root/docs/changes/archive/*)`; for each
   entry that is a directory, `Load` its `onto-state.yaml`: missing/invalid →
   finding; loaded but `!Archived` → finding. (Non-directory entries under
   `archive/` are ignored — the archive holds change directories.)

Each change/entry is keyed by its directory basename in the finding text so a
reader can locate it.

### D4 — Reuse `ValidateSkeleton` for phase-vs-artifacts

`ValidateSkeleton(changeDir)` already re-loads the state, derives the phase, and
checks `RequiredArtifacts(phase)` — precisely the "phase matches artifacts"
check. `doctor` calls it directly rather than re-implementing the artifact walk.
It re-loads the state internally; the small duplicate read is acceptable for a
diagnostic and keeps the check authoritative (single source of truth with
`onto status`).

## Risks / Trade-offs

- **Double-load of state** (once for `Validate`/`DerivePhase`/`Archived`, once
  inside `ValidateSkeleton`). Accepted: a diagnostic is not hot-path, and
  reusing `ValidateSkeleton` keeps the artifact check identical to `onto
  status`'s.
- **Finding granularity**: one change can produce multiple findings (e.g.
  invalid state AND unresolved dep). We short-circuit per change only where a
  later check depends on an earlier one succeeding (can't derive phase from an
  unloadable state), so ordering avoids nonsensical cascades.
- **Exit-code coupling to main**: relies on `main` translating a returned error
  into a non-zero exit. This is already how every other onto command signals
  failure, so no new coupling is introduced.

## Migration Plan

None — additive, new subcommand only. No existing behavior changes.

## Open Questions

None. Scope is fully determined by the dual-binary design's Doctor Boundary.
