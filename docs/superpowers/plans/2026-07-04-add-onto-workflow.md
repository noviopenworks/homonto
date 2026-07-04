---
change: add-onto-workflow
design-doc: docs/superpowers/specs/2026-07-04-onto-workflow-design.md
base-ref: 94e3f5a4a35f5a567ea6975e0e7dc79ca60a7ad6
---

# onto Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship **onto**, a self-contained markdown-only development workflow (8 skills under `content/skills/`), dogfooded via `homonto apply`, with this repo's docs migrated to the new `docs/` layout.

**Architecture:** Eight SKILL.md files implement a five-phase lifecycle (open → design → build → verify → close) plus fix/tweak presets. Per-project artifacts live in one `docs/` tree; phase state is an agent-managed `state.yaml` that the dispatcher always cross-checks against verifiable file state (files win). Distribution is a root `homonto.toml` whose `[skills] own` list makes `homonto apply` symlink each skill into `~/.claude/skills/`.

**Tech Stack:** Markdown only. No Go changes. Git (`git mv` for history-preserving moves). The existing `homonto` Go CLI is used as-is for dogfooding.

**Authoritative design:** `/home/mg/homonto/docs/superpowers/specs/2026-07-04-onto-workflow-design.md`. Every skill-authoring task below MUST start by reading the relevant design-doc sections — the plan states each skill's required sections and gates but does not restate the design's full prose; the design doc is the source of section content.

## Global Constraints

- **No Go source changes.** `go test ./...` must stay green throughout (design §Testing Strategy item 4).
- **Exactly eight skills**, names verbatim: `onto`, `onto-open`, `onto-design`, `onto-build`, `onto-verify`, `onto-close`, `onto-fix`, `onto-tweak` (design §Architecture).
- Every skill is a single file `content/skills/<name>/SKILL.md`, markdown only — **no scripts, no external CLI calls as workflow machinery** (rtk/graphify preflight checks are the only commands a skill mandates).
- **Self-containment:** `grep -rn "openspec\|comet\|docs/superpowers" content/skills/` must return zero matches when the skill set is complete (design §Testing Strategy item 2).
- **All migration moves use `git mv`** to preserve history.
- **One commit per task**, message format `type: subject` (existing repo convention), ending with the Co-Authored-By/Claude-Session trailer from the environment instructions.
- **tdd mode: direct** (confirmed decision) — no failing-test-first. Validation = dry-run walkthroughs + grep self-containment checks + `go test ./...` regression + `homonto apply` dogfood proof.
- **Bootstrap ordering (critical):** this change itself runs under comet in `openspec/changes/add-onto-workflow/`. During build, migrate everything EXCEPT the active change workspace, and keep the `openspec/` directory alive. Task 16 (retirement of `openspec/` + `docs/superpowers/`) is written now but **executed only in the comet archive phase, after the archive script has run**.
- **Evidence:** capture `./homonto doctor` and `./homonto status` output (plus `plan`/`apply` output and `ls -l ~/.claude/skills/onto*`) into `openspec/changes/add-onto-workflow/validation-notes.md` so it travels into the archive.
- `homonto apply` writes to the real `~/.claude/skills/` — this is intended (dogfood). The linker never clobbers: if a conflicting non-homonto entry exists, stop and report instead of forcing.

## File Structure

```
content/skills/
├── onto/SKILL.md           # dispatcher (Task 4)
├── onto-open/SKILL.md      # Task 5
├── onto-design/SKILL.md    # Task 6
├── onto-build/SKILL.md     # Task 7
├── onto-verify/SKILL.md    # Task 8
├── onto-close/SKILL.md     # Task 9
├── onto-fix/SKILL.md       # Task 10
└── onto-tweak/SKILL.md     # Task 10
docs/
├── adr/README.md           # ADR contract + template (Tasks 1, 3)
├── specs/README.md         # living-spec contract (Task 1) + migrated specs (Task 14)
├── changes/README.md       # workspace + state.yaml contract (Tasks 1, 2)
├── changes/archive/        # migrated archives (Task 15)
├── guides/README.md        # guides contract (Task 1)
├── guides/onto-workflow.md # user guide (Task 12)
└── roadmap.md              # migrated roadmap (Task 15)
homonto.toml                # dogfood wiring (Task 11)
README.md                   # dev-workflow section (Task 13)
```

---

## Phase 1: Foundation

### Task 1: `docs/` workflow layout skeleton (tasks.md 1.1)

**Files:**
- Create: `docs/adr/README.md`
- Create: `docs/specs/README.md`
- Create: `docs/changes/README.md`
- Create: `docs/changes/archive/.gitkeep`
- Create: `docs/guides/README.md`

**Interfaces:**
- Produces: the four directory contracts every onto skill references by path (`docs/adr/`, `docs/specs/`, `docs/changes/`, `docs/changes/archive/`, `docs/guides/`). Tasks 2 and 3 append sections to `docs/changes/README.md` and `docs/adr/README.md`.

- [x] **Step 1: Create `docs/specs/README.md`**

```markdown
# Living Capability Specs

One file per capability: `docs/specs/<capability>.md`. Each spec describes
what the system does **now** — always true, never a change log.

## Format

- `## Requirements`, containing one or more `### Requirement: <name>` blocks.
- Each requirement states a single SHALL sentence.
- Each requirement has one or more `#### Scenario: <name>` blocks written as
  **GIVEN / WHEN / THEN** bullets. Scenarios are the units the onto verify
  phase checks with fresh evidence.

## Lifecycle

- Living specs change only by merging a change's **delta spec**
  (`docs/changes/<name>/specs/<capability>.md`), which uses
  `## ADDED Requirements`, `## MODIFIED Requirements`, and
  `## REMOVED Requirements` sections.
- `onto-close` performs the merge when a change is archived: ADDED blocks are
  appended, MODIFIED blocks replace the requirement of the same name, REMOVED
  blocks are deleted. A delta for a new capability creates the spec file.
```

- [x] **Step 2: Create `docs/changes/README.md`** (state.yaml schema is added by Task 2)

```markdown
# Change Workspaces

Every unit of work is a **change** with its own workspace
`docs/changes/<name>/`. Closed changes move verbatim to
`docs/changes/archive/YYYY-MM-DD-<name>/`.

## Workspace contents

| File | Written in | Purpose |
|---|---|---|
| `state.yaml` | open (updated every phase) | agent-managed phase state — see schema below |
| `proposal.md` | open | why + what + capability impact |
| `tasks.md` | open (skeleton), build (checked off) | checklist, one commit per task |
| `design.md` | design | confirmed technical design (full workflow only) |
| `adr/<slug>.md` | design | ADR drafts, `Status: Proposed`, unnumbered |
| `specs/<capability>.md` | design | delta spec: ADDED/MODIFIED/REMOVED requirements |
| `plan.md` | build | implementation plan (full workflow) |
| `verification.md` | verify | evidence-based verification report |

A change is **active** iff its directory sits directly under `docs/changes/`
(not under `archive/`) and `state.yaml` has `archived: false` (or is absent —
the dispatcher rebuilds it).

## Archive contract

`onto-close` moves the whole workspace to
`docs/changes/archive/YYYY-MM-DD-<name>/` (date = close date), unmodified
except `archived: true` in `state.yaml`. Archived changes are history — never
edited afterwards.
```

- [x] **Step 3: Create `docs/adr/README.md`** (template + numbering are added by Task 3)

```markdown
# Architecture Decision Records

`docs/adr/` holds **accepted or superseded** decisions only, one file per
decision: `NNNN-<slug>.md`.

## Staging rule

ADRs are drafted inside a change workspace
(`docs/changes/<name>/adr/<slug>.md`) with `Status: Proposed` and **no
number**. At close, `onto-close` assigns the next free global number and
moves the draft here with `Status: Accepted`. This keeps `docs/adr/` free of
abandoned-change noise and avoids number collisions between parallel changes.
```

- [x] **Step 4: Create `docs/guides/README.md`**

```markdown
# Guides

User-facing documentation, one topic per file: `docs/guides/<topic>.md`.
Guides explain how to *use* the system; specs define what it *must do*.

## Obligation

Every onto change carries a `guides` obligation in its `state.yaml`
(`pending | updated | waived: <reason>`). `onto-close` refuses to archive a
change while `guides: pending` — either write/update the affected guide(s)
or record an explicit waiver reason.
```

- [x] **Step 5: Create `docs/changes/archive/.gitkeep`** (empty file, so the archive directory exists in git before the first migration lands)

- [x] **Step 6: Verify layout**

Run: `find docs/adr docs/specs docs/changes docs/guides -type f | sort`
Expected: exactly the five files created above.

- [x] **Step 7: Commit**

```bash
git add docs/adr docs/specs docs/changes docs/guides
git commit -m "feat: add onto docs/ layout skeleton with directory contracts"
```

### Task 2: `state.yaml` schema documentation (tasks.md 1.2)

**Files:**
- Modify: `docs/changes/README.md` (append section)

**Interfaces:**
- Consumes: `docs/changes/README.md` from Task 1.
- Produces: the canonical `state.yaml` schema + phase-derivation table that Tasks 4–10 reference (skills point to this file instead of duplicating the schema).

- [x] **Step 1: Append the schema section to `docs/changes/README.md`**

Copy the `state.yaml` example block and the phase-derivation table **verbatim from the design doc** (`docs/superpowers/specs/2026-07-04-onto-workflow-design.md`, §Architecture → "state.yaml (agent-managed)"), wrapped as:

```markdown
## state.yaml schema

```yaml
change: add-foo            # must equal directory name
workflow: full             # full | fix | tweak
phase: build               # open | design | build | verify | close
created: 2026-07-04
base_ref: <git sha at open>
decisions:                 # null until chosen (build entry)
  isolation: branch        # branch | worktree
  execution: direct        # direct | subagent
  tdd: tdd                 # tdd | direct
verify:
  mode: null               # light | full (set at verify entry by scale rules)
  result: pending          # pending | pass | fail
guides: pending            # pending | updated | waived: <reason>
archived: false
```

## Lifecycle and recovery rules

- The agent edits this file directly — there are no scripts.
- **state.yaml is a cache of truth, not truth.** Verifiable file state wins.
- On every `/onto` dispatch the phase is re-derived from artifacts using the
  table below and cross-checked; on mismatch the dispatcher corrects
  state.yaml to match the files, announces the correction to the user, and
  continues from the real state. A missing or malformed state.yaml is
  rebuilt the same way instead of failing.
- An explicit user directive that pre-answers a gate (e.g. "run to
  completion") is recorded **verbatim** under `decisions:`.

## Phase derivation (first match from bottom wins)

| Evidence | Real phase |
|---|---|
| `archived: true` or workspace under `archive/` | done |
| `verification.md` exists + `verify.result: pass` | close |
| all tasks checked in `tasks.md` | verify |
| `design.md` confirmed (or preset) + plan/tasks in progress | build |
| `proposal.md` + `tasks.md` exist, no confirmed design | design (full) / build (preset) |
| workspace exists, artifacts incomplete | open |
```

- [x] **Step 2: Verify** the appended section matches the design doc: open both files side by side and diff the yaml block and table by eye — field names, allowed values, and table rows must be identical.

- [x] **Step 3: Commit**

```bash
git add docs/changes/README.md
git commit -m "docs: document state.yaml schema, lifecycle, and phase derivation"
```

### Task 3: ADR template and numbering convention (tasks.md 1.3)

**Files:**
- Modify: `docs/adr/README.md` (append sections)

**Interfaces:**
- Consumes: `docs/adr/README.md` from Task 1.
- Produces: the ADR file template used by Task 15 (extracted ADRs 0001–0005) and referenced by `onto-design`/`onto-close` (Tasks 6, 9).

- [x] **Step 1: Append numbering + template sections**

```markdown
## Numbering

- Four digits, zero-padded, strictly increasing: `0001`, `0002`, …
- The next number = highest existing number in `docs/adr/` + 1, assigned
  **only at close** by `onto-close`. Drafts in change workspaces are
  unnumbered (`<slug>.md`).
- Numbers are never reused. A superseded ADR keeps its file; its Status
  becomes `Superseded by NNNN`.

## Template

```markdown
# <Title, imperative: "Adopt X", "Use Y for Z">

- **Status:** Proposed | Accepted | Superseded by NNNN
- **Date:** YYYY-MM-DD
- **Change:** <change name that produced this decision>

## Context

What forces are at play; why a decision is needed.

## Decision

What we decided, stated actively ("We will …").

## Consequences

What becomes easier/harder; trade-offs accepted; follow-ups.
```
```

- [x] **Step 2: Verify**

Run: `grep -n "^## Staging rule\|^## Numbering\|^## Template" docs/adr/README.md`
Expected: three hits, in that order.

- [x] **Step 3: Commit**

```bash
git add docs/adr/README.md
git commit -m "docs: add ADR template and numbering convention"
```

---

## Phase 2: Skill Set

Common rules for Tasks 4–10 (read the design doc first, every time):

- File: `content/skills/<name>/SKILL.md`, YAML frontmatter with exactly `name` and `description` (description must state what the skill does AND when to use it — this is Claude Code's trigger text).
- Every sub-skill (Tasks 5–10) is **independently loadable**: it opens with an **Entry check** section that restates its expected `phase`/`workflow` in `state.yaml`, what to do when the check fails (route back through `/onto`), so a cold session can start from any phase.
- Every sub-skill ends with an **Exit checklist** section: artifact conditions that must hold + the `state.yaml` fields to update before handing off (this replaces comet's guard scripts — design §Key Trade-offs: no hard enforcement, so exit checklists must be explicit and complete).
- Blocking decision points are marked with an unmistakable convention — use a `> **GATE:**` blockquote stating the question, the allowed answers, and the pre-authorization rule for that gate (design §Blocking decision points: gates 3 and 5 may be pre-authorized by a verbatim recorded directive; gates 1, 2, 4, 6 always need fresh input unless that same question was explicitly pre-answered).
- Reference the layout/state contracts by path (`docs/changes/README.md`, `docs/adr/README.md`, `docs/specs/README.md`) instead of duplicating them — EXCEPT the dispatcher, which embeds the phase-derivation table because it must work standalone.
- **Forbidden strings anywhere in these files:** `openspec`, `comet`, `docs/superpowers` (self-containment).

Per-task verification (run after each of Tasks 4–10):

```bash
grep -rn "openspec\|comet\|docs/superpowers" content/skills/<name>/SKILL.md
```
Expected: no matches (exit code 1).

### Task 4: `onto` dispatcher skill (tasks.md 2.1)

**Files:**
- Create: `content/skills/onto/SKILL.md`
- Read first: design doc §Architecture (skill table, state.yaml, phase derivation), §Required tooling, §GitHub entry points, §Error Handling.

**Interfaces:**
- Produces: the routing contract every sub-skill's Entry check points back to; the phase-derivation cross-check behavior Tasks 17–18 dry-run.

- [x] **Step 1: Write frontmatter**

```yaml
---
name: onto
description: onto workflow dispatcher. Use when starting, resuming, or asking about any development work in a repo with the docs/ onto layout — runs tooling preflight, finds the active change, derives the real phase from file state, and routes to the matching onto sub-skill.
---
```

- [x] **Step 2: Write the body** with these required sections (content per the design-doc sections named above — do not invent new behavior):

1. **Tooling preflight** (hard requirement, runs first, in order): (a) `rtk --version` must succeed — then all subsequent shell commands go through rtk; on failure HALT and print rtk install instructions. (b) graphify must be available (graphify skill present, or an existing `graphify-out/` or `.codegraph/` index) — open/design phases must ground codebase understanding in graphify/codegraph queries; on failure HALT and print graphify install instructions. Include the literal install-instruction text to print for each.
2. **Active-change discovery**: scan `docs/changes/*/` excluding `archive/`. Zero active changes → route to `onto-open` with the user's description. Exactly one → resume it. Two or more → list them (name, workflow, claimed phase) and ASK the user which to resume before doing anything else.
3. **Phase derivation and cross-check**: embed the phase-derivation table (copy from `docs/changes/README.md` — must match it verbatim) and the cache-of-truth rules: re-derive on every dispatch; files win on mismatch; correct `state.yaml`, announce the correction, continue from real state; missing/malformed `state.yaml` → rebuild from the table and announce (never fail).
4. **Routing table**: derived phase → skill to load: open→`onto-open`, design→`onto-design`, build→`onto-build`, verify→`onto-verify`, close→`onto-close`; `workflow: fix`→`onto-fix` and `workflow: tweak`→`onto-tweak` own their whole lifecycle (route to them for any phase of a preset change); done→report the change is archived and ask what's next.
5. **GitHub entry points (contract)**: resolve-issue seeds `onto-open` clarification from the issue text (fix preset for bugs, full for features; worktree isolation); continue-pr resumes the matching change's build phase or opens a `fix` change referencing the PR; PR creation/review stay outside — onto ends at a verified, closed change on a branch.
6. **Exit**: after routing, the dispatcher is done; the sub-skill owns the phase. Never execute phase work inside the dispatcher.

- [x] **Step 3: Verify** — run the forbidden-strings grep (expected: no matches) and `grep -n "^## " content/skills/onto/SKILL.md` (expected: the six sections above present); read the embedded derivation table against `docs/changes/README.md` (must be identical).

- [x] **Step 4: Commit**

```bash
git add content/skills/onto
git commit -m "feat: add onto dispatcher skill"
```

### Task 5: `onto-open` skill (tasks.md 2.2)

**Files:**
- Create: `content/skills/onto-open/SKILL.md`
- Read first: design doc §Architecture (skill table row `onto-open`, layout contract), §Blocking decision points (gate 1), §Design rigor.

**Interfaces:**
- Consumes: state.yaml schema (`docs/changes/README.md`), dispatcher routing (Task 4).
- Produces: workspace artifacts (`state.yaml`, `proposal.md`, `tasks.md`) whose exact shapes `onto-design`/`onto-build` consume.

- [x] **Step 1: Write frontmatter**

```yaml
---
name: onto-open
description: onto phase 1 — open a change. Use when starting a new change or when the dispatcher routes here (phase open) — clarifies requirements, checks for scope splits, and creates the change workspace with proposal and tasks skeleton.
---
```

- [x] **Step 2: Write the body** with these required sections:

1. **Entry check**: no active workspace for this work yet, or `state.yaml` says `phase: open`. Otherwise route back through `/onto`.
2. **Steps**: (a) clarify — ask questions until the requirement is unambiguous; ground every codebase claim in graphify/codegraph queries, never guesswork; (b) split preflight — if the request spans multiple independent subsystems, propose separate changes and let the user choose; (c) create `docs/changes/<name>/` with initial `state.yaml` (schema per `docs/changes/README.md`: `phase: open`, `workflow` per route, `created`, `base_ref` = current git sha, `decisions: null` fields, `verify` pending, `guides: pending`, `archived: false`), `proposal.md` (why + what + capability impact — which `docs/specs/` capabilities this touches), and `tasks.md` skeleton (unchecked checklist grouped by area).
3. **Blocking points**: GATE 1a clarification-complete confirmation; GATE 1b artifact review (user reads proposal + tasks skeleton and approves). Both always require fresh user input.
4. **Exit checklist**: workspace exists with all three artifacts; both gates answered; `phase` advanced to `design` (workflow full) or `build` (fix/tweak); announce the transition and load the next skill.

- [x] **Step 3: Verify** — forbidden-strings grep (no matches); confirm the four sections exist.

- [x] **Step 4: Commit**

```bash
git add content/skills/onto-open
git commit -m "feat: add onto-open skill"
```

### Task 6: `onto-design` skill (tasks.md 2.3)

**Files:**
- Create: `content/skills/onto-design/SKILL.md`
- Read first: design doc §Design rigor, §Blocking decision points (gate 2), §Architecture (workspace `adr/` + `specs/` deltas), brainstorm-summary decisions 1–2 (ADR staging, spec format).

**Interfaces:**
- Consumes: `proposal.md` + `tasks.md` from onto-open; ADR template (`docs/adr/README.md`); delta-spec format (`docs/specs/README.md`).
- Produces: confirmed `design.md`, `adr/<slug>.md` drafts, `specs/<capability>.md` deltas — inputs to `onto-build` and `onto-close`.

- [x] **Step 1: Write frontmatter**

```yaml
---
name: onto-design
description: onto phase 2 — deep design. Use when an active full-workflow change has phase design — brainstorming-grade exploration, approach confirmation, then design.md plus ADR drafts and spec deltas.
---
```

- [x] **Step 2: Write the body** with these required sections:

1. **Entry check**: `state.yaml` has `phase: design` and `workflow: full`; `proposal.md` exists. Presets never enter this phase (except via upgrade backfill — say so). Otherwise route back through `/onto`.
2. **Steps** (brainstorming discipline per design doc §Design rigor): explore ground truth first (graphify/codegraph queries, read the real code); question until clear; present **2–3 candidate approaches** with trade-offs; after confirmation write `design.md` (summary, goals/non-goals, architecture, error handling, testing strategy); draft ADRs for each significant decision into `docs/changes/<name>/adr/<slug>.md` (`Status: Proposed`, unnumbered, template per `docs/adr/README.md`); write delta specs into `docs/changes/<name>/specs/<capability>.md` (ADDED/MODIFIED/REMOVED requirement blocks with SHALL + Given/When/Then scenarios, format per `docs/specs/README.md`).
3. **Blocking points**: GATE 2 approach confirmation — the final `design.md` MUST NOT be written before the user confirms an approach; always fresh input. Plus the hard prohibition: **no implementation code in this phase** — writing source code before a confirmed design exists is prohibited.
4. **Exit checklist**: `design.md` marked confirmed (record confirmation date + "Status: Confirmed" line); ADR drafts and delta specs exist for every decision/spec impact named in the design; `phase: build`; announce and hand off.

- [x] **Step 3: Verify** — forbidden-strings grep (no matches); four sections present; the code-prohibition sentence exists (`grep -n "implementation code" content/skills/onto-design/SKILL.md` → at least one hit).

- [x] **Step 4: Commit**

```bash
git add content/skills/onto-design
git commit -m "feat: add onto-design skill"
```

### Task 7: `onto-build` skill (tasks.md 2.4)

**Files:**
- Create: `content/skills/onto-build/SKILL.md`
- Read first: design doc §Design rigor (writing-plans + TDD + systematic-debugging disciplines), §Blocking decision points (gate 3 + pre-authorization rule), §Error Handling (build/test failure row).

**Interfaces:**
- Consumes: confirmed `design.md`, `tasks.md`; `decisions` fields in state.yaml schema.
- Produces: `plan.md`, checked-off `tasks.md`, one commit per task — the state `onto-verify` requires.

- [x] **Step 1: Write frontmatter**

```yaml
---
name: onto-build
description: onto phase 3 — plan and build. Use when an active change has phase build — writes the implementation plan, pauses at the plan-ready gate, then executes bite-sized tasks with one commit each under the chosen TDD/direct mode.
---
```

- [x] **Step 2: Write the body** with these required sections:

1. **Entry check**: `phase: build`; for `workflow: full` a confirmed `design.md` exists (if not → back to design via `/onto`); presets enter directly after open.
2. **Steps**: (a) write `docs/changes/<name>/plan.md` — bite-sized tasks mirroring `tasks.md`, each with exact file paths, what to do, and how to verify; (b) plan-ready gate (below); (c) record execution config in `state.yaml` `decisions:` — `isolation: branch|worktree`, `execution: direct|subagent`, `tdd: tdd|direct`; (d) execute one task at a time: when `tdd: tdd`, failing test FIRST, watch it fail, minimal implementation, watch it pass; when `tdd: direct`, implement then run the task's stated verification; (e) after each task's verification passes: check it off in `tasks.md` and commit before starting the next task — never batch; (f) on ANY build/test/unexpected failure: systematic-debugging discipline — reproduce, read the actual error, form a hypothesis, find the root cause; **no source fix may be proposed before the root cause is identified**; (g) mid-build spec/design change: small edits inline, larger scope changes pause and go back through the design gate.
3. **Blocking points**: GATE 3 plan-ready + execution config — pause after `plan.md` for user review and config choice; MAY be pre-authorized by an explicit user directive recorded verbatim under `decisions:` in state.yaml, in which case proceed with the recorded config but still surface the plan summary.
4. **Exit checklist**: every `tasks.md` item checked; every task committed; no uncommitted changes; `phase: verify`; announce and hand off.

- [x] **Step 3: Verify** — forbidden-strings grep (no matches); four sections present; `grep -n "root cause" content/skills/onto-build/SKILL.md` → at least one hit.

- [x] **Step 4: Commit**

```bash
git add content/skills/onto-build
git commit -m "feat: add onto-build skill"
```

### Task 8: `onto-verify` skill (tasks.md 2.5)

**Files:**
- Create: `content/skills/onto-verify/SKILL.md`
- Read first: design doc §Design rigor (verification-before-completion), §Blocking decision points (gate 4), §Error Handling (verify fail row); delta spec "Evidence-based verification" requirement.

**Interfaces:**
- Consumes: `design.md`, delta specs, checked `tasks.md`, `verify.mode`/`verify.result` state fields.
- Produces: `verification.md` + `verify.result` — the precondition for `onto-close`.

- [x] **Step 1: Write frontmatter**

```yaml
---
name: onto-verify
description: onto phase 4 — verify. Use when an active change has phase verify (all tasks checked) — picks a verification level from change scale, checks implementation against design and every spec scenario with fresh evidence, and writes verification.md.
---
```

- [x] **Step 2: Write the body** with these required sections:

1. **Entry check**: `phase: verify` and all `tasks.md` items checked (if not, the dispatcher's derivation table sends this back to build — say so).
2. **Steps**: (a) scale check → set `verify.mode`: `full` when `workflow: full` or an upgrade occurred or the diff is large (state the rule concretely: full = full workflow, upgraded preset, or >5 files / new capability touched; light = preset within its limits); (b) check the implementation against `design.md` and **every scenario in every delta spec** — for each: run the command(s) that demonstrate the behavior, capture the actual output; (c) run the project's regression suite; (d) write `docs/changes/<name>/verification.md`: table of requirement scenario → verdict (pass/fail) → evidence (the literal command + output), plus regression results; every claim needs fresh evidence — no "should work", no stale output; (e) set `verify.result: pass|fail`.
3. **Blocking points**: GATE 4 on failure — list the failing items and ask the user: fix (→ back to build, phase reset) or accept deviation (record the deviation + rationale in `verification.md`, then result may be pass-with-deviations). Always fresh input.
4. **Exit checklist**: `verification.md` written with evidence for every scenario; `verify.result: pass` (or accepted deviations recorded); `phase: close`; announce and hand off.

- [x] **Step 3: Verify** — forbidden-strings grep (no matches); four sections present; `grep -n "evidence" content/skills/onto-verify/SKILL.md` → multiple hits.

- [x] **Step 4: Commit**

```bash
git add content/skills/onto-verify
git commit -m "feat: add onto-verify skill"
```

### Task 9: `onto-close` skill (tasks.md 2.6)

**Files:**
- Create: `content/skills/onto-close/SKILL.md`
- Read first: design doc §Architecture (layout contract, ADR staging), §Blocking decision points (gate 5), §Error Handling (guides row); delta spec "Close phase with documentation obligation" requirement.

**Interfaces:**
- Consumes: delta specs, ADR drafts, `guides` state field, `verification.md`.
- Produces: merged `docs/specs/`, numbered `docs/adr/NNNN-*.md`, updated `docs/guides/`, archived workspace.

- [x] **Step 1: Write frontmatter**

```yaml
---
name: onto-close
description: onto phase 5 — close. Use when an active change has phase close (verification passed) — merges spec deltas, numbers and accepts ADR drafts, enforces the guides obligation, and archives the workspace after final confirmation.
---
```

- [x] **Step 2: Write the body** with these required sections:

1. **Entry check**: `phase: close`, `verification.md` exists, `verify.result: pass` (or pass-with-recorded-deviations). Otherwise route back through `/onto`.
2. **Steps**: (a) merge each `docs/changes/<name>/specs/<capability>.md` delta into `docs/specs/<capability>.md` per the merge semantics in `docs/specs/README.md` (ADDED appends, MODIFIED replaces same-named requirement, REMOVED deletes; new capability → create the file); (b) for each ADR draft in the workspace `adr/`: assign the next free number (highest in `docs/adr/` + 1), move to `docs/adr/NNNN-<slug>.md`, set `Status: Accepted`; (c) guides obligation: write/update the affected `docs/guides/<topic>.md`, set `guides: updated` — or record `guides: waived: <reason>` with the user's explicit reason; archiving with `guides: pending` is prohibited; (d) final confirmation gate; (e) archive: `git mv docs/changes/<name> docs/changes/archive/YYYY-MM-DD-<name>` (today's date), set `archived: true` in the moved `state.yaml`, commit.
3. **Blocking points**: GATE 5 final confirmation before archive — summarize what was merged/numbered/documented and ask; MAY be pre-authorized by a verbatim recorded directive (still surface the summary). The guides obligation is its own hard block (design §Error Handling): close cannot complete while `guides: pending`.
4. **Exit checklist**: specs merged, ADRs numbered+accepted, guides updated or waived with reason, workspace under `archive/` with `archived: true`, everything committed; announce completion.

- [x] **Step 3: Verify** — forbidden-strings grep (no matches); four sections present; `grep -n "guides" content/skills/onto-close/SKILL.md` → hits covering the obligation.

- [x] **Step 4: Commit**

```bash
git add content/skills/onto-close
git commit -m "feat: add onto-close skill"
```

### Task 10: `onto-fix` and `onto-tweak` preset skills (tasks.md 2.7)

**Files:**
- Create: `content/skills/onto-fix/SKILL.md`
- Create: `content/skills/onto-tweak/SKILL.md`
- Read first: design doc §Architecture (skill table rows), §Blocking decision points (gate 6); delta spec "Preset paths with upgrade rules" requirement (the thresholds live there — copy them exactly).

**Interfaces:**
- Consumes: onto-open/build/verify/close section contracts (presets orchestrate those phases in lite form).
- Produces: `workflow: fix|tweak` lifecycles the dispatcher routes to.

- [x] **Step 1: Write `content/skills/onto-fix/SKILL.md`**

Frontmatter:

```yaml
---
name: onto-fix
description: onto preset — bug fix. Use for behavior fixes that need no new capability design — open-lite, then build starting from a failing test that reproduces the bug, verify, close; upgrades to the full workflow when scope grows.
---
```

Required sections:

1. **Entry check**: new bug-fix request, or active change with `workflow: fix`.
2. **Steps**: open-lite (minimal clarification — reproduce steps + expected behavior; `proposal.md` references the bug/issue; `tasks.md` skeleton; `state.yaml` with `workflow: fix`, skip design); build — **a failing test reproducing the bug is required FIRST, regardless of the tdd decision**, then root-cause per systematic-debugging discipline, then the fix, watch the test pass, commit per task; verify (light mode unless upgraded — evidence still required, scenario = the bug's reproduction); close (same obligations as `onto-close`: guides check, archive, confirmation).
3. **Upgrade rules (GATE 6)**: pause, explain the trigger, and require fresh user confirmation to upgrade to the full workflow (backfilling the design phase) when ANY of: **3+ files touched, architecture/schema changes, new public API**. On upgrade: set `workflow: full`, `phase: design`, route through `/onto`.
4. **Exit checklist** per phase (mirroring the corresponding full-skill checklists in lite form).

- [x] **Step 2: Write `content/skills/onto-tweak/SKILL.md`**

Frontmatter:

```yaml
---
name: onto-tweak
description: onto preset — small non-bug change. Use for copy, configuration, documentation, or prompt tweaks — open-lite, lightweight build, light verify, close; upgrades to the full workflow when scope grows.
---
```

Required sections:

1. **Entry check**: small non-bug change request, or active change with `workflow: tweak`.
2. **Steps**: open-lite (one-paragraph proposal + short `tasks.md`; `workflow: tweak`); lightweight build (no `plan.md` required; still one commit per task and the systematic-debugging rule on any failure); light verify (evidence for the changed behavior + regression suite; `verification.md` may be brief but never absent); close (full `onto-close` obligations).
3. **Upgrade rules (GATE 6)**: pause + fresh confirmation to upgrade when ANY of: **5+ files, cross-module coordination, 5+ new tests, config key additions/removals, a new capability, or spec-affecting changes**.
4. **Exit checklist** per phase.

- [x] **Step 3: Verify both**

```bash
grep -rn "openspec\|comet\|docs/superpowers" content/skills/onto-fix content/skills/onto-tweak
```
Expected: no matches. Also `grep -n "3+ files" content/skills/onto-fix/SKILL.md` and `grep -n "5+ files" content/skills/onto-tweak/SKILL.md` each → one hit (thresholds present exactly).

- [x] **Step 4: Commit**

```bash
git add content/skills/onto-fix content/skills/onto-tweak
git commit -m "feat: add onto-fix and onto-tweak preset skills"
```

---

## Phase 3: Integration

### Task 11: Dogfood wiring — `homonto.toml` + apply (tasks.md 3.1)

**Files:**
- Create: `homonto.toml` (repo root)
- Create: `openspec/changes/add-onto-workflow/validation-notes.md` (evidence log; lives in the active change workspace so it archives with the change)

**Interfaces:**
- Consumes: the eight `content/skills/onto*/SKILL.md` files (Tasks 4–10); the existing `homonto` CLI (`[skills] own` → `~/.claude/skills/<name>` symlinks, per `internal/adapter/claude/claude.go:151-152`).
- Produces: live symlinks that Task 19 and the comet verify phase re-check.

- [x] **Step 1: Write `homonto.toml`**

```toml
[skills]
own = [
  "onto",
  "onto-open",
  "onto-design",
  "onto-build",
  "onto-verify",
  "onto-close",
  "onto-fix",
  "onto-tweak",
]
```

- [x] **Step 2: Pre-check for clobber risk** (the linker never clobbers — resolve conflicts before apply)

Run: `ls -ld ~/.claude/skills/onto* 2>/dev/null`
Expected: no output (nothing exists yet). If entries exist and are NOT symlinks into this repo, STOP and ask the user before proceeding.

- [x] **Step 3: Build and plan**

```bash
go build -o homonto .
./homonto plan
```
Expected: plan lists a link creation for each of the eight onto skills (`~/.claude/skills/<name>` → `<repo>/content/skills/<name>`), no destructive changes, exit 0.

- [x] **Step 4: Apply and verify symlinks**

```bash
./homonto apply --yes
ls -l ~/.claude/skills/onto*
```
Expected: apply succeeds; `ls -l` shows eight symlinks, each resolving into `/home/mg/homonto/content/skills/<name>`.

- [x] **Step 5: Capture evidence** — create `openspec/changes/add-onto-workflow/validation-notes.md` with a `## Dogfood` section containing the literal output of: `./homonto plan` (from Step 3), `./homonto apply --yes`, `ls -l ~/.claude/skills/onto*`, **`./homonto status`** (expected: no drift), and **`./homonto doctor`** (expected: all checks healthy). These two command outputs are required evidence for the verify phase — do not skip them.

- [x] **Step 6: Regression**

Run: `go test ./...`
Expected: all packages pass (no Go changes were made; this proves it).

- [x] **Step 7: Commit**

```bash
git add homonto.toml openspec/changes/add-onto-workflow/validation-notes.md
git commit -m "feat: dogfood onto skills via homonto.toml (apply evidence captured)"
```

### Task 12: onto workflow guide + GitHub entry points doc (tasks.md 3.2)

**Files:**
- Create: `docs/guides/onto-workflow.md`

**Interfaces:**
- Consumes: dispatcher contract (Task 4), layout contracts (Tasks 1–3).
- Produces: the user-facing guide; also satisfies this change's own `guides` obligation.

- [x] **Step 1: Write `docs/guides/onto-workflow.md`** covering, in this order (each section is a summary that links to the authoritative file — do not duplicate contracts):

1. **What onto is** — five phases + two presets, one paragraph.
2. **Quick start** — invoke `/onto <description>` (or `/onto-fix`, `/onto-tweak`); what the dispatcher does (preflight → discovery → derive phase → route).
3. **The layout** — the `docs/` tree diagram (copy the tree from design doc §Architecture "Per-project artifact layout") with one line per directory pointing at its README.
4. **Phase walkthrough** — one short paragraph per phase naming its artifacts and its gates.
5. **Presets and upgrade rules** — when to use fix vs tweak vs full; the upgrade thresholds (same numbers as Task 10).
6. **GitHub entry points** — resolve-issue → seeds a new change (fix preset for bugs, full otherwise, worktree isolation); continue-pr → resumes the matching change's build phase or opens a fix change referencing the PR; PR creation/review remain in their own skills.
7. **Required tooling** — rtk + graphify are hard requirements; what happens when missing.

- [x] **Step 2: Verify** — `grep -n "resolve-issue\|continue-pr" docs/guides/onto-workflow.md` → hits in section 6; every relative link in the file resolves (`ls` each linked path).

- [x] **Step 3: Commit**

```bash
git add docs/guides/onto-workflow.md
git commit -m "docs: add onto workflow guide with GitHub entry points"
```

### Task 13: README development-workflow section (tasks.md 3.3)

**Files:**
- Modify: `README.md` (append a section; do not restructure existing content)

**Interfaces:**
- Consumes: `docs/guides/onto-workflow.md` (Task 12).

- [x] **Step 1: Append to `README.md`**

```markdown
## Development workflow

This repo is developed with **onto**, a self-contained markdown workflow
shipped from this very repo (`content/skills/onto*` — dogfooded via
`homonto apply`). Five phases (open → design → build → verify → close) plus
`/onto-fix` and `/onto-tweak` presets; artifacts live under `docs/`:

- `docs/adr/` — accepted architecture decisions
- `docs/specs/` — living capability specs (SHALL + scenarios)
- `docs/changes/` — active change workspaces (+ `archive/`)
- `docs/guides/` — user-facing guides

Start with `/onto`. Full guide: [docs/guides/onto-workflow.md](docs/guides/onto-workflow.md).
```

- [x] **Step 2: Verify** — `grep -n "Development workflow" README.md` → one hit; the link target exists.

- [x] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add development-workflow section to README"
```

---

## Phase 4: Migration

> **Bootstrap ordering (applies to all of Phase 4):** this change runs under
> the old machinery, in `openspec/changes/add-onto-workflow/`. Tasks 14–15
> migrate everything EXCEPT (a) the active change workspace, (b) this
> change's design doc `docs/superpowers/specs/2026-07-04-onto-workflow-design.md`
> and this plan file, and (c) the `openspec/` directory itself, which must
> stay alive because the archive step still merges this change's delta spec
> into `openspec/specs/`. Task 16 retires them and is **executed in the
> archive phase**, after the archive script has run — never during build.

### Task 14: Migrate living specs (tasks.md 4.1)

**Files:**
- Move: `openspec/specs/*/spec.md` → `docs/specs/<capability>.md` (five files, exact pairs below)

**Interfaces:**
- Consumes: `docs/specs/README.md` (Task 1) — migrated files must already satisfy its format (they do; the OpenSpec block format was kept on purpose).
- Produces: `docs/specs/<capability>.md` files that `onto-verify`/`onto-close` operate on from now on.

- [x] **Step 1: Move the five spec files with `git mv`** (flatten `<cap>/spec.md` → `<cap>.md`)

```bash
git mv openspec/specs/apply-pipeline/spec.md     docs/specs/apply-pipeline.md
git mv openspec/specs/cli-commands/spec.md       docs/specs/cli-commands.md
git mv openspec/specs/config-model/spec.md       docs/specs/config-model.md
git mv openspec/specs/secret-references/spec.md  docs/specs/secret-references.md
git mv openspec/specs/tool-adapters/spec.md      docs/specs/tool-adapters.md
```

Do NOT delete `openspec/specs/` even though it is now empty in git — the archive step will create `openspec/specs/onto-workflow/spec.md` there when it merges this change's delta (moved to `docs/` by Task 16).

- [x] **Step 2: Verify**

```bash
ls docs/specs/            # expected: README.md + the five .md files
git log --follow --oneline docs/specs/apply-pipeline.md | tail -3
```
Expected: history predates this change (follow works → `git mv` preserved it).

- [x] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: migrate living specs from openspec/specs to docs/specs"
```

### Task 15: Migrate archives, roadmap, and extract ADRs (tasks.md 4.2)

**Files:**
- Move (exact source → destination pairs from the design doc's Migration Plan — every row except the three deferred to Task 16):

| Source | Destination |
|---|---|
| `openspec/changes/archive/2026-07-03-homonto-v1-core/` (whole dir) | `docs/changes/archive/2026-07-03-homonto-v1-core/` |
| `docs/superpowers/specs/2026-07-03-homonto-v1-core-design.md` | `docs/changes/archive/2026-07-03-homonto-v1-core/design-doc.md` |
| `docs/superpowers/plans/2026-07-03-homonto-v1-core.md` | `docs/changes/archive/2026-07-03-homonto-v1-core/plan.md` |
| `docs/superpowers/reports/2026-07-03-homonto-v1-core-verify.md` | `docs/changes/archive/2026-07-03-homonto-v1-core/verification.md` |
| `docs/superpowers/specs/2026-06-24-homonto-design.md` | `docs/changes/archive/2026-06-24-homonto/design-doc.md` |
| `docs/superpowers/plans/2026-06-24-homonto.md` | `docs/changes/archive/2026-06-24-homonto/plan.md` |
| `docs/superpowers/specs/2026-07-03-homonto-roadmap.md` | `docs/roadmap.md` |

- Create: `docs/adr/0001-plan-confirm-apply-pipeline.md`, `docs/adr/0002-secrets-referenced-never-stored.md`, `docs/adr/0003-owned-content-symlinked-surgical-merge.md`, `docs/adr/0004-atomic-writes-state-last.md`, `docs/adr/0005-adopt-onto-workflow.md`

**Interfaces:**
- Consumes: ADR template (`docs/adr/README.md`, Task 3); source decisions in `docs/changes/archive/2026-07-03-homonto-v1-core/design-doc.md` (post-move) and the repo `README.md`.

- [x] **Step 1: Move the v1-core archive and its superpowers artifacts**

```bash
git mv openspec/changes/archive/2026-07-03-homonto-v1-core docs/changes/archive/2026-07-03-homonto-v1-core
git mv docs/superpowers/specs/2026-07-03-homonto-v1-core-design.md docs/changes/archive/2026-07-03-homonto-v1-core/design-doc.md
git mv docs/superpowers/plans/2026-07-03-homonto-v1-core.md docs/changes/archive/2026-07-03-homonto-v1-core/plan.md
git mv docs/superpowers/reports/2026-07-03-homonto-v1-core-verify.md docs/changes/archive/2026-07-03-homonto-v1-core/verification.md
```

- [x] **Step 2: Move the 2026-06-24 change artifacts and the roadmap**

```bash
mkdir -p docs/changes/archive/2026-06-24-homonto
git mv docs/superpowers/specs/2026-06-24-homonto-design.md docs/changes/archive/2026-06-24-homonto/design-doc.md
git mv docs/superpowers/plans/2026-06-24-homonto.md docs/changes/archive/2026-06-24-homonto/plan.md
git mv docs/superpowers/specs/2026-07-03-homonto-roadmap.md docs/roadmap.md
```

After this, `docs/superpowers/` contains ONLY this change's design doc (`specs/2026-07-04-onto-workflow-design.md`) and this plan (`plans/2026-07-04-add-onto-workflow.md`) — both deferred to Task 16. Verify: `find docs/superpowers -type f` → exactly those two files.

- [x] **Step 3: Write ADRs 0001–0004** (extracted decisions, per design doc §Key Decisions items 2–5). Use the Task 3 template; `Status: Accepted`; `Change: homonto-v1-core`; date `2026-07-03`. Source the Context/Decision/Consequences content from `docs/changes/archive/2026-07-03-homonto-v1-core/design-doc.md` and `README.md` (§Secrets, §the symlink/merge and atomic-write paragraphs):

- `0001-plan-confirm-apply-pipeline.md` — terraform-style plan → confirm → apply with tool adapters translating one desired-state model per tool.
- `0002-secrets-referenced-never-stored.md` — `${pass:...}`/`${ENV}` tokens; plan never resolves; apply resolves after confirm, all-before-any-write; state stores token + sha256 hash only.
- `0003-owned-content-symlinked-surgical-merge.md` — owned content symlinked into tools (never copied, never clobbered); non-owned keys merged surgically.
- `0004-atomic-writes-state-last.md` — temp-file+rename atomic writes; state written last so an interrupted apply leaves every file valid.

- [x] **Step 4: Write `docs/adr/0005-adopt-onto-workflow.md`** — `Status: Accepted`; `Change: add-onto-workflow`; date `2026-07-04`. Context: comet+openspec machinery (external CLI, bash guard/state scripts) vs. self-containment; Decision: markdown-only skills + agent-managed state.yaml with file-state-wins recovery (summarize design doc §Summary); Consequences: no hard guard enforcement (mitigated by exit checklists + evidence discipline), state drift possible (mitigated by derivation cross-check), portable to any repo via `homonto apply`.

- [x] **Step 5: Verify migration audit**

```bash
ls docs/adr/                                   # README.md + 0001..0005
find docs/superpowers -type f                  # exactly the 2 deferred files
find openspec -type f | grep -v add-onto-workflow   # expected: nothing
git log --follow --oneline docs/roadmap.md | tail -3  # history preserved
```

- [x] **Step 6: Commit**

```bash
git add -A
git commit -m "chore: migrate archives and roadmap to docs/, extract ADRs 0001-0005"
```

### Task 16: Retire `openspec/` and `docs/superpowers/` — EXECUTED AT ARCHIVE, NOT DURING BUILD (tasks.md 4.3)

> **DO NOT run these steps during the build phase.** This task is executed in
> the comet **archive** phase, immediately AFTER the archive script has (a)
> moved `openspec/changes/add-onto-workflow/` to
> `openspec/changes/archive/2026-07-04-add-onto-workflow/` and (b) merged the
> delta spec into `openspec/specs/onto-workflow/spec.md`. During build, mark
> this task as deferred in tasks.md (note "runs at close"; its steps below use
> the `- [>]` deferred marker) — the build exit
> checklist must not treat it as skipped.

**Files:**
- Move: `openspec/specs/onto-workflow/spec.md` → `docs/specs/onto-workflow.md`
- Move: `openspec/changes/archive/2026-07-04-add-onto-workflow/` → `docs/changes/archive/2026-07-04-add-onto-workflow/`
- Move: `docs/superpowers/specs/2026-07-04-onto-workflow-design.md` → `docs/changes/archive/2026-07-04-add-onto-workflow/design-doc.md`
- Move: `docs/superpowers/plans/2026-07-04-add-onto-workflow.md` (this plan) → `docs/changes/archive/2026-07-04-add-onto-workflow/plan.md`
- Move: this change's verify report (written by the verify phase under `docs/superpowers/reports/`) → `docs/changes/archive/2026-07-04-add-onto-workflow/verification.md`
- Delete: `openspec/` and `docs/superpowers/` (empty after the moves)

- [>] **Step 1: Confirm the archive script has run** — `ls openspec/changes/archive/ | grep add-onto-workflow` and `test -f openspec/specs/onto-workflow/spec.md`. If either fails, STOP — the ordering constraint is violated.

- [>] **Step 2: Move everything with `git mv`** (adjust the verify-report filename to whatever the verify phase actually produced under `docs/superpowers/reports/`)

```bash
git mv openspec/specs/onto-workflow/spec.md docs/specs/onto-workflow.md
git mv openspec/changes/archive/2026-07-04-add-onto-workflow docs/changes/archive/2026-07-04-add-onto-workflow
git mv docs/superpowers/specs/2026-07-04-onto-workflow-design.md docs/changes/archive/2026-07-04-add-onto-workflow/design-doc.md
git mv docs/superpowers/plans/2026-07-04-add-onto-workflow.md docs/changes/archive/2026-07-04-add-onto-workflow/plan.md
git mv docs/superpowers/reports/<verify-report>.md docs/changes/archive/2026-07-04-add-onto-workflow/verification.md
```

- [>] **Step 3: Remove the now-empty trees**

```bash
find openspec docs/superpowers -type f     # expected: nothing (git-tracked)
rm -rf openspec docs/superpowers
```

- [>] **Step 4: Dangling-reference audit**

```bash
grep -rn "openspec/\|docs/superpowers" README.md docs/ content/ .claude/ 2>/dev/null
```
Expected: no matches outside `docs/changes/archive/` (archived history may mention old paths; live files must not). Fix any live hits.

- [>] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: retire openspec/ and docs/superpowers/ (onto workflow live)"
```

---

## Phase 5: Validation

### Task 17: Dry-run walkthrough — full lifecycle (tasks.md 5.1)

**Files:**
- Create + delete: `docs/changes/dryrun-sample/` (scratch workspace, NEVER committed)
- Modify: `openspec/changes/add-onto-workflow/validation-notes.md` (append results)

- [x] **Step 1: Simulate a full lifecycle** on a scratch change `dryrun-sample` by following the eight SKILL.md files literally (agent-simulated: play both agent and user; the "user" answers each gate). Walk open → design → build → verify → close and check off each item of this checklist as it is observed:

1. dispatcher preflight runs first (`rtk --version` succeeds; graphify present);
2. zero-active-changes → routed to `onto-open`;
3. GATE 1a (clarification) and GATE 1b (artifact review) both stop for input;
4. workspace created with `state.yaml`/`proposal.md`/`tasks.md` matching `docs/changes/README.md` exactly;
5. `phase` advances open→design; GATE 2 blocks `design.md` until approach confirmed; ADR draft + delta spec land in workspace `adr/` and `specs/`;
6. GATE 3 plan-ready pause; `decisions:` recorded; task check-off + commit-per-task rule stated for each task;
7. `phase` build→verify only when all tasks checked; `verification.md` demands per-scenario evidence; `verify.result` set;
8. close: delta merge semantics, ADR numbering (next free number), guides obligation blocks until updated/waived, GATE 5, archive move + `archived: true`;
9. a fresh `/onto` dispatch at every phase boundary derives the same phase the state claims (cross-check consistent).

Any step where a skill is ambiguous or contradicts the design doc → fix that SKILL.md now (that is the point of the dry run) and note the fix.

- [x] **Step 2: Record results** — append a `## Dry-run: full lifecycle` section to `openspec/changes/add-onto-workflow/validation-notes.md`: the checklist above with pass marks, plus any skill fixes made.

- [x] **Step 3: Clean up** — `rm -rf docs/changes/dryrun-sample` and `git status` must show no trace of the scratch workspace.

- [x] **Step 4: Commit**

```bash
git add openspec/changes/add-onto-workflow/validation-notes.md content/skills
git commit -m "test: full-lifecycle dry-run walkthrough of onto skills"
```

### Task 18: Dry-run presets + drift recovery (tasks.md 5.2 + design testing item 1d)

**Files:**
- Create + delete: `docs/changes/dryrun-fix/`, `docs/changes/dryrun-tweak/` (scratch, never committed)
- Modify: `openspec/changes/add-onto-workflow/validation-notes.md` (append results)

- [x] **Step 1: `/onto-fix` walkthrough** on scratch change `dryrun-fix`: open-lite skips design; failing-test-first is demanded before any fix; then simulate the fix growing to touch **four files** → the skill must hit GATE 6, explain the "3+ files" trigger, and require fresh confirmation; on confirmed upgrade, `workflow: full` + `phase: design` backfill is prescribed.

- [x] **Step 2: `/onto-tweak` walkthrough** on scratch change `dryrun-tweak`: lightweight build without `plan.md`; light verify still produces `verification.md`; close still enforces the guides obligation. Confirm at least one tweak upgrade trigger fires correctly when simulated (e.g. a config key addition).

- [x] **Step 3: Drift recovery**: in `dryrun-tweak`, hand-corrupt `state.yaml` to `phase: verify` while `tasks.md` has unchecked tasks, then simulate a fresh `/onto` dispatch — the dispatcher's derivation table must reset the phase to build, announce the correction, and resume. Also delete `state.yaml` entirely and dispatch again — it must be rebuilt from the table, not fail.

- [x] **Step 4: Record + clean up** — append `## Dry-run: presets + drift` results to validation-notes.md; `rm -rf docs/changes/dryrun-fix docs/changes/dryrun-tweak`; `git status` clean of scratch dirs.

- [x] **Step 5: Commit**

```bash
git add openspec/changes/add-onto-workflow/validation-notes.md content/skills
git commit -m "test: preset and drift-recovery dry-runs for onto skills"
```

### Task 19: Self-containment, symlink load, regression (tasks.md 5.3)

**Files:**
- Modify: `openspec/changes/add-onto-workflow/validation-notes.md` (append results)

- [x] **Step 1: Self-containment grep**

```bash
grep -rn "openspec\|comet\|docs/superpowers" content/skills/
```
Expected: **no matches** (exit code 1). Any hit is a defect — fix the skill and re-run.

- [x] **Step 2: Symlink load check**

```bash
ls -l ~/.claude/skills/onto*
test -f ~/.claude/skills/onto/SKILL.md && echo RESOLVES
head -5 ~/.claude/skills/onto/SKILL.md
```
Expected: eight symlinks into `/home/mg/homonto/content/skills/`; `RESOLVES`; the dispatcher frontmatter prints. (Human check, note for the user: in a fresh Claude Code session `/onto` should appear in the available-skills list.)

- [x] **Step 3: Status/doctor re-check + regression**

```bash
./homonto status    # expected: no drift
./homonto doctor    # expected: healthy
go test ./...       # expected: all pass
```
Capture all three outputs verbatim into validation-notes.md under `## Final checks` (fresh evidence for the verify phase).

- [x] **Step 4: Migration audit (build-scope)**

```bash
grep -rn "openspec/specs\|docs/superpowers" README.md docs/guides docs/specs docs/adr content/
```
Expected: no matches (live docs reference only new paths; the deferred Task 16 paths live only in this plan and the change workspace).

- [x] **Step 5: Commit**

```bash
git add openspec/changes/add-onto-workflow/validation-notes.md
git commit -m "test: self-containment, symlink, and regression evidence for onto"
```

---

## Task → tasks.md mapping

| Plan task | tasks.md item |
|---|---|
| 1, 2, 3 | 1.1, 1.2, 1.3 |
| 4–10 | 2.1–2.7 |
| 11, 12, 13 | 3.1, 3.2, 3.3 |
| 14, 15, 16 | 4.1, 4.2, 4.3 (16 executes at archive) |
| 17, 18, 19 | 5.1, 5.2, 5.3 |

Check each tasks.md box in `openspec/changes/add-onto-workflow/tasks.md` as its plan task's commit lands (mark 4.3 as "deferred to close" during build).
