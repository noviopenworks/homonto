# Delta Spec: onto-workflow (polish-onto-framework)

## ADDED Requirements

### Requirement: Artifact templates

Every onto artifact SHALL have a canonical template shipped as a reference
file inside the skill that creates it (`onto/references/state-yaml.md`,
`onto-open/references/{proposal,tasks,notes}.md`,
`onto-design/references/{design,adr-draft,delta-spec}.md`,
`onto-build/references/{plan,subagent-protocol}.md`,
`onto-verify/references/{verification,adversarial}.md`,
`onto-close/references/{lint-checklist,ship-handoff}.md`); skills instruct
when to read each template, and artifacts that deviate from their
template's structure are lint findings at close.

#### Scenario: Artifact created from template

- **GIVEN** onto-open creating a new change's proposal
- **WHEN** the skill is followed
- **THEN** it reads `references/proposal.md` and produces a proposal with
  that template's exact section structure

#### Scenario: Missing reference file degrades, never halts

- **GIVEN** a skill whose `references/` directory is unavailable
- **WHEN** the skill needs a template
- **THEN** it reconstructs the artifact from the `docs/changes/README.md`
  pointers, notes the gap, and continues

### Requirement: Context-loss checkpoints

Each full-workflow change SHALL keep an incremental `notes.md` checkpoint
(created at open, template-based) recording confirmed facts, candidate
decisions, gate answers, and *pending* items — presets SHOULD create one
for work spanning sittings. onto-open and onto-design SHALL update it
before ending any turn that produced new decisions, and every phase skill
SHALL read it at entry when present. The derivation table recovers
*where* a change is; notes.md recovers *why* — and state rebuild consults
its Confirmed gate answers before crossing any phase boundary.

#### Scenario: Compaction during design

- **GIVEN** a design conversation lost to context compaction before
  `design.md` exists
- **WHEN** `/onto` re-dispatches into the design phase
- **THEN** onto-design resumes from `notes.md`'s confirmed facts and
  pending items instead of re-asking answered questions

### Requirement: Parallel-change coordination

`state.yaml` SHALL support a `deps:` list naming changes that must archive
before this change may build; the dispatcher SHALL show each active
change's deps status during discovery and SHALL warn — requiring an
explicit user choice — before resuming a change whose deps are not all
archived. For multiple simultaneously active changes the workflow SHALL
recommend one worktree per change.

#### Scenario: Blocked dependency

- **GIVEN** change B with `deps: [change-a]` while change-a is not archived
- **WHEN** the user resumes change B
- **THEN** the dispatcher warns and asks: proceed anyway, switch to
  change-a, or stop

## MODIFIED Requirements

### Requirement: Artifact layout contract

The workflow SHALL keep all artifacts in a single `docs/` tree: numbered
ADRs in `docs/adr/`, living capability specs in `docs/specs/`, per-change
workspaces in `docs/changes/<name>/`, closed changes in
`docs/changes/archive/YYYY-MM-DD-<name>/`, and user-facing guides in
`docs/guides/`. Workspace artifacts SHALL follow the canonical templates
bundled with the skills.

#### Scenario: Change workspace contents

- **GIVEN** a full-workflow change past the design phase
- **WHEN** its workspace is inspected
- **THEN** it contains `state.yaml`, `proposal.md`, `notes.md`,
  `design.md`, `tasks.md`, and (as produced) `adr/` drafts, `specs/`
  deltas, `plan.md`, and `verification.md`, each matching its template's
  structure

### Requirement: Agent-managed state with file-state recovery

Each change SHALL have a `state.yaml` (change, workflow, phase, created,
base_ref, deps, decisions, verify, guides, metrics, archived) that the
agent edits directly. Verifiable file state SHALL be the source of truth:
on every dispatch the phase is re-derived from artifacts, and on mismatch
the dispatcher corrects `state.yaml`, announces the correction, and
continues from the real state. Metrics SHALL be best-effort observational
data — stamped at phase exits, never a gate, and never blocking during
rebuild.

#### Scenario: Corrupted state file

- **GIVEN** a change whose `state.yaml` is missing or malformed
- **WHEN** `/onto` dispatches
- **THEN** the dispatcher rebuilds `state.yaml` from the phase-derivation
  table and per-field rules (deps from the proposal's `Depends-on:` line
  else empty; metrics from phase-advance commit dates else omitted) and
  announces the correction instead of failing

#### Scenario: State claims a later phase than files support

- **GIVEN** `state.yaml` says `phase: verify` but `tasks.md` has unchecked
  tasks
- **WHEN** `/onto` dispatches
- **THEN** the dispatcher resets the phase to build and resumes execution

### Requirement: Build discipline

The build phase SHALL produce an implementation plan with bite-sized tasks,
execute one task at a time with one commit per task, require a failing test
first for each task when `tdd: tdd`, and require root-cause analysis
(systematic-debugging discipline) before any fix when a build, test, or
unexpected failure occurs; parallel subagent dispatch, when used,
preserves one commit per task through per-implementer worktrees with
coordinator-performed serial joins. When `execution: subagent`, the main
session SHALL act only as coordinator: one fresh-context implementer
agent per task (given the task, exact files, design section, conventions,
and verification), file-based checkoffs and one commit per task verified
by the coordinator against the repository, and a fault-finding reviewer
agent after any high-risk task and the final task.

#### Scenario: Task completion

- **GIVEN** an in-progress build phase
- **WHEN** a task's verification passes
- **THEN** the task is checked off in `tasks.md` and committed before the
  next task starts

#### Scenario: Coordinator never implements

- **GIVEN** `execution: subagent` recorded in `state.yaml`
- **WHEN** a task needs code written
- **THEN** an implementer agent is dispatched for it, and the coordinator
  verifies the resulting commit and checkoffs in the repository rather
  than trusting the agent's report

### Requirement: Evidence-based verification

The verify phase SHALL select a verification level from change scale, check
the implementation against the design and every delta-spec scenario, and
write `verification.md` containing fresh command output as evidence for
each claim. In full mode the phase SHALL additionally dispatch two
fresh-context skeptic agents in parallel — conformance (attempt to refute
each scenario claim) and robustness (edge cases, drift and recovery paths)
— and triage their findings into the report; in light mode one skeptic is
optional with any skip recorded. If no subagent capability exists, the
skipped adversarial pass SHALL be recorded in the report's Adversarial
section (protocol-mandated skips need no acceptor). A failed verification
SHALL block close until the user chooses fix or accept-deviation
(recorded in the report).

#### Scenario: Verification pass

- **GIVEN** all tasks complete and checks pass with recorded evidence
- **WHEN** the verify phase completes
- **THEN** `verify.result: pass` is written and the workflow may enter close

#### Scenario: Skeptic refutes a claim

- **GIVEN** a conformance skeptic demonstrating that a scenario's evidence
  does not hold
- **WHEN** findings are triaged
- **THEN** that scenario's verdict becomes fail and the verify failure gate
  applies

### Requirement: Close phase with documentation obligation

The close phase SHALL lint the change before merging (delta-spec format:
only ADDED/MODIFIED/REMOVED/RENAMED sections, SHALL/MUST in every
ADDED/MODIFIED requirement's first line, GIVEN/WHEN/THEN scenarios;
state.yaml validity; `Result:` line present; ADR draft fields; post-merge
no delta-only section headings; dangling-reference audit) with findings
blocking archive exactly like the guides obligation; then merge spec deltas into `docs/specs/`
(including RENAMED semantics), assign final numbers to ADR drafts and move
them to `docs/adr/` with status Accepted, write or update `docs/guides/`
(or record an explicit `guides: "waived: <reason>"`), finalize
`metrics` (phase dates, tasks_total, verify_rounds, upgraded), archive the
workspace to `docs/changes/archive/YYYY-MM-DD-<name>/`, set
`archived: true` — after final user confirmation — and offer a ship
handoff: a ready PR body (proposal why/what + verification summary +
evidence pointers) written to the archive's `ship.md` when accepted, with
PR creation remaining outside the workflow.

#### Scenario: Guides not updated

- **GIVEN** a change reaching close with `guides: pending`
- **WHEN** the agent attempts to archive
- **THEN** close is blocked until guides are updated or a waiver reason is
  recorded

#### Scenario: Malformed delta caught at lint

- **GIVEN** a delta spec whose requirement lacks SHALL in its first line
- **WHEN** close runs the lint checklist
- **THEN** the finding blocks the merge until the delta is fixed

#### Scenario: RENAMED requirement merged

- **GIVEN** a delta with `## RENAMED Requirements` mapping FROM/TO names
- **WHEN** close merges the delta
- **THEN** the living spec's requirement heading is renamed with its body
  preserved unless a MODIFIED block also targets the new name
