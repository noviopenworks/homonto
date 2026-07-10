# Comet Design Handoff

- Change: add-onto-workflow
- Phase: design
- Mode: compact
- Context hash: 61026599da4b65da70e825a6f26a2c16d37839a0ec8a642821eb2b3683691bf4

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/add-onto-workflow/proposal.md

- Source: openspec/changes/add-onto-workflow/proposal.md
- Lines: 1-73
- SHA256: 6456fc80e315cf4d56f394d1ede4483d7e461b634c7487a96a95a03a28724713

```md
# Proposal: add-onto-workflow

## Why

homonto development currently runs on the Comet workflow, which depends on
external machinery: the `openspec` npm CLI, comet bash guard/state scripts, and
a split artifact layout (`openspec/` + `docs/superpowers/`). That machinery is
not portable, not self-describing, and not aligned with how this project wants
to document itself (ADRs, living capability specs, and user-facing guides
written after implementation).

homonto's own roadmap (v1.1) calls for shipping curated built-in content. A
homonto-native workflow skill set — authored as homonto-owned content and
dogfooded through `homonto apply` — solves both problems at once: a
self-contained development workflow for this and future projects, and the first
real entry in the template catalog.

## What Changes

- Add the **onto** workflow skill set as homonto-owned content under
  `content/skills/`: `/onto` (dispatcher), `/onto-open`, `/onto-design`,
  `/onto-build`, `/onto-verify`, `/onto-close`, plus `/onto-fix` and
  `/onto-tweak` presets with upgrade rules.
- Workflow is fully self-contained: no `openspec` CLI, no shell guard scripts.
  Phase state lives in a tiny agent-managed `state.yaml` per change; verifiable
  file state is the source of truth on conflict.
- Introduce the docs-centralized artifact layout: `docs/adr/` (numbered ADRs),
  `docs/specs/` (living capability specs), `docs/changes/` (active change
  workspaces + archive), `docs/guides/` (mandatory post-implementation docs).
- Keep Comet-grade rigor: mandatory brainstorming-style design (except
  presets), TDD-first build with commit-per-task, verification report before
  close, explicit user-confirmation blocking points.
- Hard-require `rtk` (token-optimized shell ops) and `graphify` (codebase
  understanding during open/design); the workflow halts with install
  instructions when either is missing.
- Document `resolve-issue` / `continue-pr` GitHub skills as entry points into
  onto; PR creation/review stay outside the workflow.
- Wire the skills into `homonto.toml` (`[skills]` owned content) so
  `homonto apply` symlinks them into `.claude/skills/` (dogfood).
- **Migrate** this repo's existing `openspec/` and `docs/superpowers/`
  artifacts into the new `docs/` layout; onto replaces Comet for homonto
  development going forward.

## Capabilities

### New Capabilities

- `onto-workflow`: the onto development workflow — phase model and dispatch,
  artifact layout contract (adr/specs/changes/guides), state model and
  resume/recovery rules, preset paths and upgrade rules, required tooling
  (rtk, graphify), GitHub entry points, and close-phase documentation
  obligations.

### Modified Capabilities

<!-- none — no homonto binary behavior (apply-pipeline, cli-commands,
config-model, secret-references, tool-adapters) changes in this change -->

## Impact

- **New files**: `content/skills/onto*/SKILL.md` (8 skills), `docs/adr/`,
  `docs/specs/`, `docs/changes/`, `docs/guides/` trees.
- **Moved/merged files**: `openspec/specs/*` → `docs/specs/`;
  `openspec/changes/archive/*` → `docs/changes/archive/`;
  `docs/superpowers/{specs,plans,reports}/*` → ADRs, archived change records,
  or `docs/guides/` as appropriate; `openspec/` and `docs/superpowers/`
  directories retired.
- **Modified files**: `homonto.toml` (own the onto skills), `README.md`
  (development workflow note), `.claude/` (symlinks created by
  `homonto apply`).
- **No Go source changes**; homonto binary behavior is untouched.
- **Dependencies**: none added to the product; the workflow itself assumes
  `rtk` and `graphify` are installed on the developer machine.
```

## openspec/changes/add-onto-workflow/design.md

- Source: openspec/changes/add-onto-workflow/design.md
- Lines: 1-74
- SHA256: 41413a556f53558f0644c5177cf7a85ef2f9f96ba94d07d3ff33de4760b81ad8

```md
# Design (high-level): add-onto-workflow

Deep technical design happens in the design phase (Design Doc + delta spec).
This file records the open-phase architecture decisions and approach selection.

## Approach Selection

Considered:

1. **Fork Comet** (copy skills + scripts, swap paths) — rejected: keeps the
   bash guard/state machinery and openspec CLI dependency we want to remove.
2. **Pure convention, no state file** — rejected by user: phase should be
   explicit and cheap to read; deriving it purely from artifact presence makes
   resume/edge cases ambiguous.
3. **Markdown-only skills + tiny agent-managed state file** — **chosen**: all
   workflow logic lives in SKILL.md prose the agent follows; a small
   `state.yaml` per change records phase and decisions; verifiable file state
   overrides it on conflict. Nothing to install, nothing to execute.

## Architecture

```
content/skills/                     # authored here (homonto-owned)
├── onto/SKILL.md                   # dispatcher: detect phase → route
├── onto-open/SKILL.md              # clarify → proposal/design/tasks
├── onto-design/SKILL.md            # brainstorm → design doc + ADR drafts + spec deltas
├── onto-build/SKILL.md             # plan → TDD tasks → commit per task
├── onto-verify/SKILL.md            # checks vs design/specs → verification.md
├── onto-close/SKILL.md             # merge spec deltas, accept ADRs, write guides, archive
├── onto-fix/SKILL.md               # preset: bugfix (skips design; upgrade rules)
└── onto-tweak/SKILL.md             # preset: small change (skips design+full plan)

homonto.toml [skills] own → homonto apply → symlinks into .claude/skills/

docs/                               # workflow artifact layout (per project)
├── adr/NNNN-<title>.md             # numbered ADRs (proposed → accepted at close)
├── specs/<capability>.md           # living capability specs (deltas merged at close)
├── changes/<name>/                 # active change workspace
│   ├── state.yaml                  # agent-managed: phase, workflow, decisions
│   ├── proposal.md  design.md  tasks.md  verification.md
│   ├── adr/ specs/                 # drafts/deltas staged for close-phase merge
├── changes/archive/YYYY-MM-DD-<name>/
└── guides/<topic>.md               # post-implementation user docs (close phase)
```

## Key Decisions

- **State**: `state.yaml` is a cache of truth, not truth. Every dispatch
  cross-checks it against artifact presence/content; mismatch → correct the
  file, continue from real state.
- **Blocking points preserved**: artifact review (open), approach confirmation
  (design), plan-ready + execution-config (build), fail handling (verify),
  final confirmation (close) — via the platform's question tool.
- **Presets**: `/onto-fix`, `/onto-tweak` skip design; upgrade rules (file
  count, architecture impact, new capability) force the full path.
- **Tooling**: rtk wraps shell ops; graphify (or its codegraph index) is the
  mandated exploration tool in open/design. Both hard-required.
- **GitHub**: resolve-issue / continue-pr are documented entry points that
  start or resume an onto change; PR creation/review remain separate skills.
- **Docs obligation**: close phase cannot complete without updating
  `docs/guides/` (or explicitly recording why no guide change is needed).

## Data Flow

issue/PR/idea → open (proposal, tasks skeleton) → design (design doc, ADR
drafts, spec deltas) → build (plan, TDD commits) → verify (verification.md)
→ close (spec merge, ADR accept, guides, archive) → done.

## Migration (this repo)

`openspec/specs/*` → `docs/specs/`; archived change → `docs/changes/archive/`;
`docs/superpowers/specs|plans|reports` → ADR extraction + archived change
records; retire `openspec/` and `docs/superpowers/`. Comet remains installed
globally but homonto development uses onto from the next change onward.
```

## openspec/changes/add-onto-workflow/tasks.md

- Source: openspec/changes/add-onto-workflow/tasks.md
- Lines: 1-55
- SHA256: b88afe2aa91af363b574df73be00aa1930069a3c12a1545e0d804e29ad8d5967

```md
# Tasks: add-onto-workflow

## 1. Foundation

- [ ] 1.1 Create `docs/` workflow layout skeleton (`adr/`, `specs/`,
      `changes/`, `changes/archive/`, `guides/`) with a README in each
      explaining its contract
- [ ] 1.2 Define the `state.yaml` schema and document it (fields, lifecycle,
      file-state-wins recovery rule)
- [ ] 1.3 Define the ADR template and numbering convention

## 2. Skill Set

- [ ] 2.1 Author `content/skills/onto/SKILL.md` — dispatcher: phase detection
      from state.yaml + file cross-check, routing table, resume rules,
      rtk/graphify preflight, GitHub entry-point contract
- [ ] 2.2 Author `content/skills/onto-open/SKILL.md` — clarification,
      split preflight, proposal/design/tasks creation, review blocking point
- [ ] 2.3 Author `content/skills/onto-design/SKILL.md` — brainstorming-grade
      design, Design Doc, ADR drafts, spec deltas, approach confirmation
- [ ] 2.4 Author `content/skills/onto-build/SKILL.md` — implementation plan,
      plan-ready pause, TDD/direct modes, commit-per-task, failure handling
- [ ] 2.5 Author `content/skills/onto-verify/SKILL.md` — verification levels,
      checks vs design/specs/tasks, verification.md, fail decision point
- [ ] 2.6 Author `content/skills/onto-close/SKILL.md` — spec delta merge, ADR
      status finalization, docs/guides obligation, archive, final confirmation
- [ ] 2.7 Author `content/skills/onto-fix/SKILL.md` and
      `content/skills/onto-tweak/SKILL.md` — preset paths + upgrade rules

## 3. Integration

- [ ] 3.1 Wire onto skills into `homonto.toml` `[skills]` and run
      `homonto apply` to symlink into `.claude/skills/` (dogfood proof)
- [ ] 3.2 Document GitHub entry points (resolve-issue / continue-pr → onto)
      in the dispatcher skill and a `docs/guides/onto-workflow.md` guide
- [ ] 3.3 Update `README.md` with the development-workflow section

## 4. Migration

- [ ] 4.1 Migrate `openspec/specs/*` → `docs/specs/` (flatten to
      `<capability>.md`)
- [ ] 4.2 Migrate archived change + `docs/superpowers/` artifacts →
      `docs/changes/archive/` and extract ADR-worthy decisions → `docs/adr/`
- [ ] 4.3 Retire `openspec/` and `docs/superpowers/` (this change's own
      workspace moves last, at close)

## 5. Validation

- [ ] 5.1 Dry-run walkthrough: simulate a full onto lifecycle on a scratch
      change (open→close) verifying each blocking point, state transition,
      and artifact contract
- [ ] 5.2 Dry-run preset paths: /onto-fix and /onto-tweak including an
      upgrade-rule trigger
- [ ] 5.3 Verify skills load via symlink (`/onto` visible to Claude Code) and
      contain no references to openspec CLI or comet scripts
```

## openspec/changes/add-onto-workflow/specs/onto-workflow/spec.md

- Source: openspec/changes/add-onto-workflow/specs/onto-workflow/spec.md
- Lines: 1-202
- SHA256: 8cdf5b9a11db77dff81ac07245ae882e7a36f73a1d3e9f64cb5f455df3ae41e4

[TRUNCATED]

```md
# Delta Spec: onto-workflow

## ADDED Requirements

### Requirement: Phase model and dispatch

The onto workflow SHALL provide a five-phase lifecycle (open → design →
build → verify → close) driven by a `/onto` dispatcher that detects the
current phase and routes to the matching sub-skill, plus `/onto-fix` and
`/onto-tweak` preset paths that skip the design phase.

#### Scenario: No active change

- **GIVEN** a repo with the onto layout and no directory under
  `docs/changes/` (other than `archive/`)
- **WHEN** the user invokes `/onto` with a change description
- **THEN** the dispatcher routes to `onto-open`, which clarifies
  requirements and creates a new change workspace

#### Scenario: Resume mid-lifecycle

- **GIVEN** an active change whose `state.yaml` says `phase: build`
- **WHEN** the user invokes `/onto` in a fresh session
- **THEN** the dispatcher cross-checks file state, confirms or corrects the
  phase, and resumes from the next unchecked task without repeating
  completed phases

#### Scenario: Multiple active changes

- **GIVEN** two or more active change workspaces
- **WHEN** the user invokes `/onto` without naming one
- **THEN** the dispatcher lists the active changes and asks the user which
  to resume before proceeding

### Requirement: Artifact layout contract

The workflow SHALL keep all artifacts in a single `docs/` tree: numbered
ADRs in `docs/adr/`, living capability specs in `docs/specs/`, per-change
workspaces in `docs/changes/<name>/`, closed changes in
`docs/changes/archive/YYYY-MM-DD-<name>/`, and user-facing guides in
`docs/guides/`.

#### Scenario: Change workspace contents

- **GIVEN** a full-workflow change past the design phase
- **WHEN** its workspace is inspected
- **THEN** it contains `state.yaml`, `proposal.md`, `design.md`, `tasks.md`,
  and (as produced) `adr/` drafts, `specs/` deltas, `plan.md`, and
  `verification.md`

### Requirement: Agent-managed state with file-state recovery

Each change SHALL have a `state.yaml` (change, workflow, phase, created,
base_ref, decisions, verify, guides, archived) that the agent edits
directly. Verifiable file state SHALL be the source of truth: on every
dispatch the phase is re-derived from artifacts, and on mismatch the
dispatcher corrects `state.yaml`, announces the correction, and continues
from the real state.

#### Scenario: Corrupted state file

- **GIVEN** a change whose `state.yaml` is missing or malformed
- **WHEN** `/onto` dispatches
- **THEN** the dispatcher rebuilds `state.yaml` from the phase-derivation
  table and announces the correction instead of failing

#### Scenario: State claims a later phase than files support

- **GIVEN** `state.yaml` says `phase: verify` but `tasks.md` has unchecked
  tasks
- **WHEN** `/onto` dispatches
- **THEN** the dispatcher resets the phase to build and resumes execution

### Requirement: Design rigor gates

The full workflow SHALL enforce blocking user-confirmation points:
clarification + artifact review (open), approach confirmation before the
final design is written (design), plan-ready + execution configuration
(build), fix-vs-accept decision on verification failure, and final
confirmation before archive (close). An explicit user directive to run
```

Full source: openspec/changes/add-onto-workflow/specs/onto-workflow/spec.md
