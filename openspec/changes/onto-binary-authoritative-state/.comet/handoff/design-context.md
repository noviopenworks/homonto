# Comet Design Handoff

- Change: onto-binary-authoritative-state
- Phase: design
- Mode: compact
- Context hash: 1629b30ba5d3ce5af40a66ac0fb5f57c630facbadb42ad49b5fb45e37f99cf65

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-binary-authoritative-state/proposal.md

- Source: openspec/changes/onto-binary-authoritative-state/proposal.md
- Lines: 1-61
- SHA256: 6697a8819276554b993d872413a9592161b1d1fe6f0e7ac910623d394d872cc6

```md
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

```

## openspec/changes/onto-binary-authoritative-state/design.md

- Source: openspec/changes/onto-binary-authoritative-state/design.md
- Lines: 1-75
- SHA256: 778c8c3c0537d4ee187b82ac338dd2019cce33a75c51d50d643f93e44095b74c

```md
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

```

## openspec/changes/onto-binary-authoritative-state/tasks.md

- Source: openspec/changes/onto-binary-authoritative-state/tasks.md
- Lines: 1-41
- SHA256: 51f87d650b89791a52b41c72f901d683969cd4fdae59e447b731bb2be6494d1f

```md
# Tasks — onto-binary-authoritative-state

Open-phase task outline. The design phase resolves the deferred decisions; the
build phase turns these into a detailed plan.

## 1. Versioned state schema
- [ ] Define the versioned schema (superset of gated control fields + carried
      observational fields) with an explicit `schema_version`.
- [ ] Split gated vs observational fields per the B1 line; validate presence/shape
      of gated fields only.
- [ ] Round-trip (marshal→parse) tests including every gated field.

## 2. Migration from both legacy shapes
- [ ] Loader recognizes legacy `onto-state.yaml` (7-field), legacy
      `docs/changes/<name>/state.yaml` (rich), and the new versioned schema.
- [ ] Ordered, idempotent up-migration to the current version on read; writes
      always emit current version.
- [ ] Conflict policy for a dir carrying both legacy files.
- [ ] Migration tests over real `state.yaml` fixtures — assert no gated field is
      dropped.

## 3. CLI transition + read surface
- [ ] Every state mutation the skills do by hand has a binary command (extend
      `init/new/advance/close`; add the missing gated-field transitions).
- [ ] Structured (JSON) read command so callers query state without parsing files.
- [ ] Tests per command (happy path + validation rejection).

## 4. status/doctor enumerate + classify
- [ ] Enumerate change directories first, then classify valid / malformed /
      missing-state.
- [ ] A deleted state file appears as a `missing-state` row (F14 regression test).

## 5. Spec + verification
- [ ] Update `openspec/specs/onto-binary/spec.md` (delta) for the versioned
      schema, command surface, and classify behavior.
- [ ] `go test ./internal/ontostate/... ./internal/ontocli/...` green under -race.
- [ ] `go build ./...`, `go vet`, `openspec validate --all` green.

## 6. Confirm change B is ready to author
- [ ] Record the final schema + CLI surface so `onto-skills-shell-out` (change B)
      can be authored against concrete commands.

```

## openspec/changes/onto-binary-authoritative-state/specs/onto-binary/spec.md

- Source: openspec/changes/onto-binary-authoritative-state/specs/onto-binary/spec.md
- Lines: 1-172
- SHA256: 7015676c4a8731be67690f0774edd203398dc47076af5c07870b42e964ef1fea

[TRUNCATED]

```md
# onto-binary (delta)

## MODIFIED Requirements

### Requirement: onto-state.yaml change-state model

The `onto` binary SHALL read, validate, and write a per-change state file named
`onto-state.yaml` (at `docs/changes/<name>/onto-state.yaml`) through a dedicated
state package, as the single authority for onto workflow state. The model SHALL
parse the file into a typed structure carrying an explicit `schema_version`, a
typed **core** of gated fields, and a carried **observational** group that is
never gated. It SHALL validate the presence and shape of gated fields only
(enum/format), never their substantive quality (B1: the binary rejects a
malformed value, not an unconvincing one). It SHALL derive the current workflow
phase from the core.

The gated core SHALL include at least: change, workflow (`full|fix|tweak`), phase
(`open|design|build|verify|close`), created, base_ref, deps, isolation
(`branch|worktree|""`), build_mode (`direct|subagent|""`), tdd_mode
(`tdd|direct|""`), verify scale (`light|full|""`), verify result
(`pending|pass|fail`), close merged (bool), archived (bool), and the directive
string. Observational fields (metrics, task counts, verify rounds, escalation
flag) SHALL be carried through reads and writes but SHALL never gate a
transition. Writes SHALL always emit the current `schema_version`.

The binary SHALL migrate legacy state on read: a legacy binary `onto-state.yaml`
(no `schema_version`) and a legacy skill `state.yaml` (no `schema_version`) SHALL
each up-migrate to the current schema. Migration SHALL be ordered and idempotent
(loading a current-version file is a no-op). If a change directory holds BOTH a
legacy `onto-state.yaml` and a legacy `state.yaml` whose gated core fields
disagree (phase, workflow, or archived), the state SHALL be reported as malformed
rather than silently resolved. Parsing an invalid or malformed state SHALL return
a clear error identifying the file, not a panic.

The recognized workflow phases are `open`, `design`, `build`, `verify`, `close`,
with `close` as the terminal phase and `archived` as a terminal boolean.

#### Scenario: parse and derive phase from a valid versioned onto-state.yaml

- **GIVEN** a valid `onto-state.yaml` carrying `schema_version`, a gated core, and observational fields
- **WHEN** the state model loads it
- **THEN** it returns the typed state and the derived phase without error, preserving observational fields

#### Scenario: legacy state migrates on read

- **GIVEN** a legacy `onto-state.yaml` (no `schema_version`) or a legacy `state.yaml` (no `schema_version`)
- **WHEN** the state model loads it
- **THEN** it up-migrates to the current schema without dropping any gated field, and a subsequent write emits the current `schema_version`

#### Scenario: disagreeing dual legacy files are malformed

- **GIVEN** a change directory holding both a legacy `onto-state.yaml` and a legacy `state.yaml` whose phase, workflow, or archived disagree
- **WHEN** the state model loads the change
- **THEN** it reports the state as malformed and names the conflict, and does not silently pick a winner

#### Scenario: malformed state reports a clear error

- **GIVEN** a state file that is not valid YAML or fails presence/shape validation
- **WHEN** the state model loads it
- **THEN** it returns an error naming the file and the problem, and does not panic

### Requirement: onto status is read-only and config-independent

`onto status` SHALL be a read-only diagnostic command that inspects an existing
`docs/` workspace WITHOUT requiring a `homonto.toml` file or a declared
`[frameworks.onto]` entry. It SHALL enumerate change **directories** under
`docs/changes/` (excluding `archive/`) FIRST, then classify each as `valid`
(state present, parses, validates — report its derived phase), `malformed` (state
present but unparseable/invalid), or `missing-state` (a change directory with no
state file). A change directory whose state file was deleted SHALL therefore
appear as a `missing-state` row and SHALL NOT silently disappear. `onto status`
SHALL NOT create, modify, or delete any file.

#### Scenario: status classifies each change directory

- **GIVEN** `docs/changes/` with one valid change, one whose `onto-state.yaml` is malformed, and one directory with no state file
- **WHEN** `onto status` runs
- **THEN** it reports the first as `valid` with its phase, the second as `malformed`, and the third as `missing-state`, and exits without writing any file

#### Scenario: a deleted state file is not silently dropped

```

Full source: openspec/changes/onto-binary-authoritative-state/specs/onto-binary/spec.md
