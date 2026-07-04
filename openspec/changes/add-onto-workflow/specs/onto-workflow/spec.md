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
autonomously MAY pre-answer the build and close gates and SHALL be recorded
verbatim in `state.yaml`; clarification, approach confirmation, verify-fail,
and preset-upgrade gates SHALL always require fresh user input unless the
user explicitly pre-answered that same question.

#### Scenario: Design cannot be skipped

- **GIVEN** a full-workflow change in the design phase
- **WHEN** the agent attempts to write implementation code
- **THEN** the skill prohibits it until a confirmed design exists

#### Scenario: Pre-authorized autonomous run

- **GIVEN** the user has explicitly directed "run to completion" and the
  directive is recorded in `state.yaml`
- **WHEN** the build phase reaches the plan-ready gate
- **THEN** the workflow proceeds with the recorded configuration and
  surfaces the plan summary instead of pausing

### Requirement: Build discipline

The build phase SHALL produce an implementation plan with bite-sized tasks,
execute one task at a time with one commit per task, require a failing test
first for each task when `tdd: tdd`, and require root-cause analysis
(systematic-debugging discipline) before any fix when a build, test, or
unexpected failure occurs.

#### Scenario: Task completion

- **GIVEN** an in-progress build phase
- **WHEN** a task's verification passes
- **THEN** the task is checked off in `tasks.md` and committed before the
  next task starts

### Requirement: Evidence-based verification

The verify phase SHALL select a verification level from change scale, check
the implementation against the design and every delta-spec scenario, and
write `verification.md` containing fresh command output as evidence for each
claim. A failed verification SHALL block close until the user chooses fix or
accept-deviation (recorded in the report).

#### Scenario: Verification pass

- **GIVEN** all tasks complete and checks pass with recorded evidence
- **WHEN** the verify phase completes
- **THEN** `verify.result: pass` is written and the workflow may enter close

### Requirement: Close phase with documentation obligation

The close phase SHALL merge spec deltas into `docs/specs/`, assign final
numbers to ADR drafts and move them to `docs/adr/` with status Accepted,
write or update `docs/guides/` (or record an explicit
`guides: waived: <reason>`), archive the workspace to
`docs/changes/archive/YYYY-MM-DD-<name>/`, and set `archived: true` — after
final user confirmation.

#### Scenario: Guides not updated

- **GIVEN** a change reaching close with `guides: pending`
- **WHEN** the agent attempts to archive
- **THEN** close is blocked until guides are updated or a waiver reason is
  recorded

### Requirement: Preset paths with upgrade rules

The workflow SHALL provide preset paths — `/onto-fix` (bug fixes; failing
test reproducing the bug required first) and `/onto-tweak`
(copy/config/docs/prompt changes) — that skip the design phase but keep
verify and close. The workflow SHALL force an upgrade confirmation
to the full path when scope grows: fix — 3+ files, architecture/schema
changes, new public API; tweak — 5+ files, cross-module coordination, 5+ new
tests, config key additions/removals, new capability, or spec-affecting
changes.

#### Scenario: Upgrade trigger

- **GIVEN** an active `/onto-fix` change
- **WHEN** the fix grows to touch four files
- **THEN** the skill pauses, explains the trigger, and asks the user to
  confirm upgrading to the full workflow before continuing

### Requirement: Required tooling preflight

The dispatcher SHALL verify `rtk` is installed (all shell operations then go
through rtk) and graphify is available (open/design phases ground codebase
understanding in graphify/codegraph queries), halting with install
instructions when either is missing.

#### Scenario: Missing rtk

- **GIVEN** a machine without rtk on PATH
- **WHEN** `/onto` is invoked
- **THEN** the workflow halts and prints rtk install instructions instead of
  proceeding

### Requirement: GitHub entry points

The workflow SHALL document `resolve-issue` and `continue-pr` as entry
points: an issue seeds a new change (fix preset for bugs, full otherwise);
PR feedback resumes the matching change's build phase or opens a fix change
referencing the PR. PR creation and review SHALL remain outside the
workflow.

#### Scenario: Issue as entry point

- **GIVEN** a GitHub issue describing a bug
- **WHEN** the user enters the workflow via resolve-issue
- **THEN** a `/onto-fix` change is opened whose proposal references the
  issue

### Requirement: Dogfood distribution

The onto skills SHALL live in `content/skills/` as homonto-owned content,
listed under `[skills] own` in the repo's `homonto.toml`, and be projected
into `~/.claude/skills/` by `homonto apply` as symlinks.

#### Scenario: Apply links skills

- **GIVEN** the repo's `homonto.toml` owning the eight onto skills
- **WHEN** the user runs `homonto apply` and confirms
- **THEN** `~/.claude/skills/onto*` are symlinks into the repo's
  `content/skills/` and `homonto status` reports no drift
