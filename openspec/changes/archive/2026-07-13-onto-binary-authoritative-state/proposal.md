## Why

onto has two incompatible control planes wearing one name. The Go binary
(`internal/ontostate`, `internal/ontocli`) owns `onto-state.yaml` with 7 fields
and no version; the markdown skill framework owns a ~20-field
`docs/changes/<name>/state.yaml`. Neither reads the other, so they can report a
different phase, workflow, and archive state (F1, F3), and a change whose state
file was deleted can vanish from `status`/`doctor` entirely (F14).

The 2026-07-13 strategic fork resolved this: onto is **binary-authoritative**
with **B1** enforcement (the binary enforces the *presence and shape* of
well-formed state/evidence, trusting the agent's judgment behind it) under a
**T-honest** threat model. This change builds the foundation that makes the
binary the single authority — one versioned schema, migration from both legacy
files, a CLI surface covering every transition, and diagnostics that never lose a
workspace. It is a hard prerequisite for the skills to stop writing state by hand
(change `onto-skills-shell-out`, created next) and for the semantic gates (N2).

## What Changes

- **One authoritative, versioned state schema** owned by the binary: a superset
  covering the workflow-control fields the skills need (isolation, build_mode,
  tdd_mode, verify scale/result, close progress, decisions.directive, deps, spec
  sync, best-effort metrics) plus an explicit `schema_version`. **BREAKING** for
  on-disk state: existing files are migrated, not read as-is.
- **Migration** from both legacy shapes — the binary's `onto-state.yaml` (7
  fields) and the skill's `docs/changes/<name>/state.yaml` (rich) — into the one
  versioned schema, on one canonical path/name (chosen in design).
- **CLI surface covering every transition and a read command**, so a caller can
  drive the whole lifecycle without editing state files (the commands the skills
  will invoke in change B).
- **`status`/`doctor` enumerate change directories first, then classify** each as
  `valid` / `malformed` / `missing-state`, so a workspace never silently
  disappears (F14).

## Capabilities

### New Capabilities

- (none — extends the existing binary capability)

### Modified Capabilities

- `onto-binary`: the state model gains a versioned superset schema and migration;
  the command surface gains full transition + read coverage; `status`/`doctor`
  change from "enumerate state files" to "enumerate change dirs, then classify."

## Impact

- **Code:** `internal/ontostate` (schema, version, migration, validation),
  `internal/ontocli` (transition + read commands, `status`/`doctor` classify).
- **Spec:** `openspec/specs/onto-binary/spec.md`.
- **Out of scope (this change):** rewriting the 9 `onto*` skills to shell out and
  deleting the "markdown-only / no external CLI" copy — that is change
  `onto-skills-shell-out` (depends on this). Making gates *semantic* (confirmed
  design, scenario coverage, `Result: pass`, merged deltas, workflow-aware
  transitions, dep resolver) is **N2**, a later change. No homonto-engine work.
- **Design decisions deferred to the design phase (brainstorming):** exact schema
  shape (full-rich vs core+typed-extension for observational fields), the one
  canonical state path/filename, reconciling the binary's terminal `close` phase
  with the skill's `close`→`archived`, and the migration/versioning mechanics.
