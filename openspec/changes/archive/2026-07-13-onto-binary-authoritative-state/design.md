## High-level approach

Make the binary the single source of truth for onto workflow state, in four
layers. Deep design (schema field-by-field, migration edge cases, command
surface) is refined in the design phase; this frames the architecture and the
decisions that phase must close.

### 1. One versioned state schema (binary-owned)

Today `internal/ontostate.State` has 7 fields and no version; the skill schema
has ~20. The binary schema becomes the **superset** the skills need, plus
`schema_version`. B1 draws the line: fields the binary must *gate on* (phase,
workflow, isolation, build_mode, tdd_mode, verify result, close progress, deps,
decisions.directive, spec-sync status) are first-class and validated for
presence/shape; purely observational fields (metrics, counts) are carried but
never gate. `schema_version` lets later changes migrate deterministically.

**Design decision:** full-rich single struct vs a typed core + an
extension/observational bag. Leaning core+typed so observational drift can't
break gating.

### 2. Migration from both legacy shapes

A loader that recognizes: (a) the binary's current `onto-state.yaml` (7 fields,
no version), (b) the skill's `docs/changes/<name>/state.yaml` (rich, no version),
and (c) the new versioned schema. Legacy inputs migrate up to the current version
on read; write always emits the current version. Migration is ordered and
idempotent.

**Design decisions:** the one canonical path + filename (reconcile
`onto-state.yaml` vs `docs/changes/<name>/state.yaml`, and which directory change
workspaces live in); how to handle a directory that carries *both* legacy files
(conflict policy); whether migration is on-read-only or a one-time `onto migrate`.

### 3. CLI surface for every transition + a read command

The skills must be able to drive the entire lifecycle without touching a state
file (change B consumes this). Extend `internal/ontocli` so every state
mutation the skills perform today by hand has a command, plus a structured read
(so a skill can query state). Reuse the existing `init/new/advance/close`
where they fit; add what's missing (e.g. setting isolation/build_mode/verify
result/close progress) as explicit subcommands or a guarded `set`.

**Design decision:** a small set of semantic transition commands vs a general
`onto state set <field> <value>` with validation. Leaning semantic transitions
for the gated fields (so the binary owns the rules), with reads structured
(JSON) for callers.

### 4. status/doctor: enumerate dirs, then classify

Invert today's "enumerate `onto-state.yaml` files": first enumerate change
**directories**, then classify each as `valid` (parses + validates),
`malformed` (present but unparseable/invalid), or `missing-state` (a change dir
with no state file). A deleted state file becomes a reported `missing-state`
row, never a silent disappearance (F14).

### Phase vocabulary

Reconcile the binary's terminal `close` with the skill's `close`→`archived`
flag. Likely: keep phases `open|design|build|verify|close` and model `archived`
as a boolean on the terminal phase (as the skill already does), so both planes
agree. Confirmed in design.

## Non-goals

- Rewriting the `onto*` skills / deleting the markdown-only copy (change B).
- Semantic gate content, workflow-aware transitions, dep resolver (N2).
- Any homonto-engine or non-onto spec change.

## Risks

- **Migration data loss:** a bad migration could drop skill-only fields. Mitigate
  with round-trip tests over real `state.yaml` fixtures before any write path.
- **Schema churn vs N2:** N2 will add gate semantics; keep the schema and command
  surface shaped so N2 extends, not rewrites (hence `schema_version` now).
