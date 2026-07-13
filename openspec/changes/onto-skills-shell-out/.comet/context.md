# Comet Design Handoff

- Change: onto-skills-shell-out
- Phase: design
- Mode: compact
- Context hash: 79620e1354b3453be8bb6b385d35aa84501665e4d9d98fc42a425666ac6cc80b

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-skills-shell-out/proposal.md

- Source: openspec/changes/onto-skills-shell-out/proposal.md
- Lines: 1-66
- SHA256: ad63da037e972a1d3a431673ef823b0320a68a775334b232a275eeb5f9587c6a

```md
## Why

Change A (`onto-binary-authoritative-state`, archived) made the onto Go binary the
single authority for onto workflow state. But the markdown skills still write
state by hand — the `onto*` skills edit `docs/changes/<name>/state.yaml`
directly, and `onto/SKILL.md` still claims onto is "markdown-only" with "no
external CLI." Both are now false. Until the skills stop hand-writing state, the
two planes can still diverge (F1/F3 residue on the skill side).

Grounding change A's surface against what the skills actually write revealed that
A's CLI does **not** cover every field: `onto new` hardcodes `workflow: full` and
sets no `base_ref`/`deps` (`internal/ontocli/new.go:79`), there is no command for
the skill's gated `guides` field (which change A never put in the schema), and the
observational `metrics` fields are carried but have no setter. So the skills
cannot shell out for everything yet. Per the 2026-07-13 scope decision, change B
**bundles** the minimal binary extensions with the skill rewrite.

## What Changes

- **Close the binary CLI-surface gaps (additive; no schema redesign):**
  - `onto new --workflow full|fix|tweak` (stop hardcoding `full`), so the
    fix/tweak presets can create their workflow via the binary.
  - `onto set base-ref <change> <ref>` and `onto set deps <change> <names...>`
    for the fields `onto new` captures today by hand.
  - Add `guides` (pending|updated|`waived: <reason>`) to the state schema as a
    gated core field, plus `onto set guides <change> <value>`.
  - **Observational fields decision (design):** either add minimal setters
    (`onto set metric <phase> <date>`, counts) or drop onto's observational
    tracking (metrics/tasks_total/verify_rounds/preset_escalated are never gated).
    Leaning **drop** — they never gate and re-derive cheaply.
- **Rewrite the `onto*` skills to shell out — zero direct state writes.** Every
  state mutation in onto, onto-open, onto-design, onto-build, onto-verify,
  onto-close, onto-fix, onto-tweak becomes an `onto <command>` invocation
  (`new`/`advance`/`close`/`set …`); reads use `onto state <change> --json` /
  `onto status`. `onto-no-slop` is prose-only and is expected to touch no state.
- **BREAKING (doctrine):** delete the "markdown-only / no external CLI" copy from
  the skills; state onto's hard dependency on the compiled binary.

## Capabilities

### New Capabilities

- (none)

### Modified Capabilities

- `onto-binary`: additive command surface (`onto new --workflow`, `onto set
  base-ref|deps|guides`), and the state-model requirement gains the `guides`
  gated field (+ the observational drop/keep outcome).

## Impact

- **Code:** `internal/ontocli/new.go` (`--workflow` flag), `internal/ontocli/set.go`
  (base-ref/deps/guides setters), `internal/ontostate/state.go` (`guides` field +
  validation; observational drop if chosen) + tests.
- **Catalog:** the 8 state-writing `onto*` skills (`catalog/skills/onto*`) rewritten
  to invoke the CLI; markdown-only copy deleted.
- **Spec:** `openspec/specs/onto-binary/spec.md` delta.
- **Out of scope:** semantic gate *content* / workflow-aware *transition rules* /
  dep resolver (N2 — note `--workflow` here only sets the field, it does not add
  workflow-aware gating); homonto-engine work (gate B); any further schema
  redesign beyond adding `guides`.
- **Design decisions deferred to the design phase:** the observational drop-vs-keep
  call; whether every one of the 8 skills fully maps to commands with no residual
  hand-write; how `deps` is passed (repeatable flag vs comma list); whether a
  thin skill capability spec is warranted or the delta stays on `onto-binary`.

```

## openspec/changes/onto-skills-shell-out/design.md

- Source: openspec/changes/onto-skills-shell-out/design.md
- Lines: 1-69
- SHA256: 0b33b44d25824d66b0b80c6003bad1c92a6c68227aaa0029eda185ccfe96b352

```md
## High-level approach

Two layers, sequenced: finish the binary command+schema surface, then rewrite the
skills against it. Deep design (per-skill command mapping, observational
drop-vs-keep, exact flag shapes) is refined in the design phase.

### Layer 1 — close the CLI-surface gaps (additive)

Building on change A's schema, add only what the skills need to stop hand-writing
state:

- **`onto new --workflow full|fix|tweak`** — `new.go` currently hardcodes
  `Workflow: "full"`. Add a validated flag. This sets the field only; it adds no
  workflow-aware transition *rules* (that is N2).
- **`onto set base-ref <change> <ref>`**, **`onto set deps <change> <names…>`** —
  the two creation fields `onto new` doesn't set. `deps` shape (repeatable flag vs
  comma list) decided in design.
- **`guides` gated field** — add to `ontostate.State` as a gated core field with
  shape `pending|updated|waived: <reason>` (the skill's contract), plus
  `onto set guides <change> <value>`. This is a small schema addition (a
  `schema_version` bump is likely unnecessary — the field is additive and
  legacy-tolerant like the others; confirm in design).
- **Observational fields** — decision point. Lean **drop**: remove
  `metrics/tasks_total/verify_rounds/preset_escalated` from onto's model since
  they never gate a transition and the skills can stop tracking them. Alternative:
  keep and add thin setters. The drop simplifies both planes.

### Layer 2 — rewrite the skills to shell out

For each of the 8 state-writing `onto*` skills, replace every "edit state.yaml"
instruction with the corresponding `onto` command:

| Skill action | Command |
|---|---|
| create change (+ workflow) | `onto new <name> --workflow <w>` |
| capture base_ref / deps | `onto set base-ref`, `onto set deps` |
| advance phase | `onto advance <name>` |
| record isolation/exec/tdd/directive | `onto set isolation|build-mode|tdd-mode|directive` |
| record verify scale/result | `onto set verify-scale|verify-result` |
| mark close.merged / guides | `onto set close-merged|guides` |
| archive | `onto close <name>` |
| read current state | `onto state <name> --json` / `onto status` |

Then **delete the "markdown-only / no external CLI" copy** from `onto/SKILL.md`
and any sibling, and state the hard binary dependency. `onto-no-slop` is
prose-discipline and should need no change (verify).

### Verification strategy

- Binary extensions: TDD, same shape as change A (happy + shape-reject per
  command; `--workflow` validation; `guides` shape).
- Skills: a **grep-based gate** proving no `onto*` skill contains a direct state
  write (no `state.yaml`/`onto-state.yaml` edit instruction) and no
  "markdown-only / no external CLI" copy — the enforceable form of the exit gate.
  A full-lifecycle skill dry-run belongs to N7 (the onto E2E suite), not here.

## Non-goals

- Semantic gate content, workflow-aware transition *rules*, dep resolver (N2).
- Homonto-engine / projection work (gate B).
- Any schema redesign beyond adding `guides` (and the observational drop).

## Risks

- **A skill writes a field with no command even after Layer 1** — mitigated by an
  explicit per-skill field→command audit in design before any rewrite.
- **Dropping observational loses history other tooling reads** — check no skill or
  doctor path depends on `metrics` before dropping; if any does, keep + add a
  setter instead.

```

## openspec/changes/onto-skills-shell-out/tasks.md

- Source: openspec/changes/onto-skills-shell-out/tasks.md
- Lines: 1-51
- SHA256: 42b070f71179bb9e094793dd7287c05a2f57c9ceedc19424ad612e71de433bda

```md
# Tasks — onto-skills-shell-out

Open-phase outline. The design phase resolves the deferred decisions (observational
drop-vs-keep, per-skill field→command audit, flag shapes); the build phase turns
these into a detailed plan.

## 1. Binary: workflow at creation
- [ ] `onto new --workflow full|fix|tweak` (validated); stop hardcoding `full`.
- [ ] Tests: each workflow accepted; bad value rejected.

## 2. Binary: creation-field setters
- [ ] `onto set base-ref <change> <ref>`.
- [ ] `onto set deps <change> <names…>` (flag shape decided in design).
- [ ] Tests: happy path + reads back.

## 3. Binary: guides field + setter
- [ ] Add `guides` gated field to `ontostate.State` (shape `pending|updated|
      waived: <reason>`) with validation; confirm whether a schema_version bump
      is needed.
- [ ] `onto set guides <change> <value>`.
- [ ] Round-trip + shape-reject tests.

## 4. Binary: observational decision
- [ ] Design-confirmed drop OR keep+setters. If drop: remove
      metrics/tasks_total/verify_rounds/preset_escalated from the model after
      confirming no skill/doctor path depends on them; update tests.

## 5. Per-skill field→command audit (design gate)
- [ ] Enumerate every state write in each of onto, onto-open, onto-design,
      onto-build, onto-verify, onto-close, onto-fix, onto-tweak and map it to a
      command. Confirm no residual field lacks one after tasks 1–4.

## 6. Rewrite skills to shell out
- [ ] Replace every direct state-write instruction in the 8 skills with the mapped
      `onto` command; reads via `onto state --json` / `onto status`.
- [ ] Confirm `onto-no-slop` needs no change.

## 7. Delete the markdown-only / no-external-CLI copy
- [ ] Remove the "markdown-only" / "no external CLI" claims from `onto/SKILL.md`
      and any sibling; state the hard binary dependency.

## 8. Enforcement gate + verification
- [ ] Grep-based CI gate: no `onto*` skill contains a direct state-file write and
      none contains the markdown-only/no-CLI copy.
- [ ] `openspec/specs/onto-binary/spec.md` delta for the added commands + guides.
- [ ] `go test ./internal/ontostate/... ./internal/ontocli/... -race`, `go vet`,
      `go build`, `openspec validate --all` green.

## 9. Out of scope (recorded)
- [ ] (note only) workflow-aware transition *rules*, semantic gates, dep resolver
      → N2; full-lifecycle skill dry-run → N7 onto E2E suite.

```

## openspec/changes/onto-skills-shell-out/specs/onto-binary/spec.md

- Source: openspec/changes/onto-skills-shell-out/specs/onto-binary/spec.md
- Lines: 1-137
- SHA256: e67b15c82ea68b19d3b4dab1babb44115bb79cc87864a0f7e21b855029e013ea

[TRUNCATED]

```md
# onto-binary (delta)

## MODIFIED Requirements

### Requirement: onto new creates a change skeleton

`onto new <change-name> [--workflow full|fix|tweak]` SHALL create
`docs/changes/<change-name>/` containing an `onto-state.yaml` (`change` = the
name, `workflow` = the `--workflow` value defaulting to `full`, `phase` = `open`,
`created` = the current date) and empty-but-present `proposal.md` and `tasks.md`
skeleton files. `--workflow` SHALL accept only `full`, `fix`, or `tweak`; any
other value SHALL be rejected with a non-zero exit and no writes. It SHALL run the
framework-install gate first (same as `onto init`), SHALL validate `<change-name>`
is kebab-case with no path traversal (reject `..`, `/`, empty), and SHALL REFUSE
with a non-zero exit and NO writes if `docs/changes/<change-name>/` already exists.

#### Scenario: new creates the open-phase skeleton with the chosen workflow

- **GIVEN** a prepared workspace (framework-install gate passes) with no `docs/changes/feature-x/`
- **WHEN** `onto new feature-x --workflow fix` runs
- **THEN** `docs/changes/feature-x/onto-state.yaml` exists with `phase: open` and `workflow: fix`, alongside `proposal.md` and `tasks.md`, exiting 0

#### Scenario: new defaults workflow to full

- **WHEN** `onto new feature-y` runs with no `--workflow`
- **THEN** the created `onto-state.yaml` has `workflow: full`

#### Scenario: new rejects an invalid workflow

- **WHEN** `onto new feature-z --workflow epic` runs
- **THEN** it exits non-zero with a validation error and creates nothing

#### Scenario: new refuses to clobber an existing change

- **GIVEN** `docs/changes/feature-x/` already exists (with content)
- **WHEN** `onto new feature-x` runs
- **THEN** it exits non-zero, prints that the change already exists, and modifies no file under `docs/changes/feature-x/`

#### Scenario: new rejects an invalid change name

- **WHEN** `onto new "../evil"` (or a non-kebab-case / empty name) runs
- **THEN** it exits non-zero with a validation error and creates nothing

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
(`pending|pass|fail`), close merged (bool), guides
(`pending|updated|"waived: <reason>"|""`), archived (bool), and the directive
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

```

Full source: openspec/changes/onto-skills-shell-out/specs/onto-binary/spec.md
