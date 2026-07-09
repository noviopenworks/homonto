# Comet Development Workflow Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Comet the active development workflow for this repository while preserving Onto history and avoiding claims that unimplemented Homonto framework projection already works.

**Architecture:** Bootstrap OpenSpec and Comet as checked-in project workflow structure, then project Comet/OpenSpec/Superpowers skills through Homonto's existing local skill mechanism. Living docs will point future agents to Comet; old `docs/changes/` Onto workspaces remain historical and are not rewritten.

**Tech Stack:** Go 1.23, Homonto local skill projection, OpenSpec CLI, Comet skill scripts, Markdown workflow docs, TOML config.

## Global Constraints

- Do not implement framework/catalog projection in this change.
- Do not claim `[frameworks.comet]` installs anything until catalog projection exists.
- Do not rewrite archived `docs/changes/archive/*` workspaces.
- Do not delete `onto` as a product/framework concept unless a separate product decision explicitly says so.
- Use local/project-scoped skill resources in `homonto.toml` for the immediate dogfood layer.
- Keep `docs/superpowers/specs/` and `docs/superpowers/plans/` as Superpowers HOW surfaces.
- Make OpenSpec the WHAT surface for new Comet-managed changes, but do not bulk-convert existing `docs/specs/*.md` in this bootstrap change.
- Commit after each task.
- Run `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `go run . status`, and `go run . doctor` before declaring the migration complete.

---

## File Structure

- Create: `.comet/config.yaml` - Comet project defaults.
- Create: `openspec/changes/` - OpenSpec active change workspaces.
- Create: `openspec/specs/` - OpenSpec canonical WHAT specs for new changes.
- Modify: `homonto.toml` - replace Onto dogfood skill declarations with Comet/OpenSpec/Superpowers local skills.
- Create: `homonto/skills/comet*/` - local Comet skill content copied from installed Comet skills.
- Create: `homonto/skills/openspec-*/` - local OpenSpec skill content copied from installed OpenSpec skills.
- Create: `homonto/skills/<superpowers-skill>/` - local Superpowers skill dependencies needed by Comet.
- Keep: `homonto/skills/onto*/` - historical/product source content remains in the repo, but is no longer projected for internal development unless intentionally re-enabled.
- Modify: `README.md` - contributor workflow starts with `/comet`.
- Modify: `docs/NEXT_AGENT.md` - Comet/OpenSpec state is the first workflow check.
- Modify: `docs/guides/README.md` - link the Comet workflow guide and mark Onto as legacy or product-scoped.
- Create: `docs/guides/comet-workflow.md` - operational guide for this repo's new workflow.
- Create: `docs/specs/comet-workflow.md` - living requirements spec for the Comet workflow.
- Modify: `docs/changes/README.md` - mark legacy Onto workspace/archive contract.
- Modify: `docs/guides/onto-workflow.md` and `docs/specs/onto-workflow.md` - mark legacy internal workflow or product-framework documentation, not current development workflow.
- Modify: `docs/road-to-release.md`, `docs/roadmap.md`, `docs/release-checklist.md`, and `docs/superpowers/specs/2026-07-09-dual-binary-release-design.md` only where needed to preserve the distinction between internal Comet development and product framework scope.

---

### Task 1: Bootstrap OpenSpec And Comet Project State

**Files:**
- Create: `.comet/config.yaml`
- Create: `openspec/changes/`
- Create: `openspec/specs/`

**Interfaces:**
- Consumes: OpenSpec CLI available on `PATH`.
- Produces: An empty OpenSpec workspace where `openspec list --json --no-color` returns `{"changes":[]}`, plus Comet defaults readable by Comet skills.

- [ ] **Step 1: Initialize OpenSpec structure**

Run:

```bash
openspec init --tools none .
```

Expected output includes:

```text
OpenSpec structure created
OpenSpec Setup Complete
```

- [ ] **Step 2: Create Comet project config**

Create `.comet/config.yaml` with exactly:

```yaml
language: en
context_compression: off
auto_transition: true
```

- [ ] **Step 3: Verify OpenSpec is initialized**

Run:

```bash
openspec list --json --no-color
```

Expected output:

```json
{"changes":[]}
```

- [ ] **Step 4: Verify Comet scripts are locatable**

Run:

```bash
COMET_ENV="${COMET_ENV:-$(find . "$HOME"/.*/skills "$HOME/.config" "$HOME/.gemini" -path '*/comet/scripts/comet-env.mjs' -type f -print -quit 2>/dev/null)}"
test -n "$COMET_ENV"
COMET_SCRIPTS_DIR="$(node "$COMET_ENV")"
test -n "$COMET_SCRIPTS_DIR"
test -f "$COMET_SCRIPTS_DIR/comet-state.mjs"
test -f "$COMET_SCRIPTS_DIR/comet-guard.mjs"
```

Expected: no output and exit code 0.

- [ ] **Step 5: Commit bootstrap structure**

Run:

```bash
git add .comet openspec
git commit -m "chore: bootstrap openspec and comet config"
```

---

### Task 2: Project Comet Through Local Homonto Skills

**Files:**
- Modify: `homonto.toml`
- Create: `homonto/skills/comet/`
- Create: `homonto/skills/comet-open/`
- Create: `homonto/skills/comet-design/`
- Create: `homonto/skills/comet-build/`
- Create: `homonto/skills/comet-verify/`
- Create: `homonto/skills/comet-archive/`
- Create: `homonto/skills/comet-hotfix/`
- Create: `homonto/skills/comet-tweak/`
- Create: `homonto/skills/openspec-*/`
- Create: `homonto/skills/brainstorming/`
- Create: `homonto/skills/writing-plans/`
- Create: `homonto/skills/executing-plans/`
- Create: `homonto/skills/subagent-driven-development/`
- Create: `homonto/skills/using-git-worktrees/`
- Create: `homonto/skills/test-driven-development/`
- Create: `homonto/skills/systematic-debugging/`
- Create: `homonto/skills/verification-before-completion/`
- Create: `homonto/skills/finishing-a-development-branch/`
- Create: `homonto/skills/requesting-code-review/`
- Create: `homonto/skills/receiving-code-review/`
- Create: `homonto/skills/dispatching-parallel-agents/`

**Interfaces:**
- Consumes: Installed source skills under `/home/mg/.claude/skills/` and `/home/mg/.agents/skills/`.
- Produces: Local project-scoped skill resources that `homonto apply` can symlink into Claude Code and OpenCode.

- [ ] **Step 1: Copy Comet skill directories**

Run:

```bash
for name in comet comet-open comet-design comet-build comet-verify comet-archive comet-hotfix comet-tweak; do
  test -d "/home/mg/.claude/skills/$name"
  rm -rf "homonto/skills/$name"
  cp -a "/home/mg/.claude/skills/$name" "homonto/skills/$name"
done
```

Expected: no output and each `homonto/skills/comet*/SKILL.md` exists.

- [ ] **Step 2: Copy OpenSpec skill directories**

Run:

```bash
for src in /home/mg/.claude/skills/openspec-*; do
  name="$(basename "$src")"
  test -d "$src"
  rm -rf "homonto/skills/$name"
  cp -a "$src" "homonto/skills/$name"
done
```

Expected: no output and at least these files exist:

```text
homonto/skills/openspec-explore/SKILL.md
homonto/skills/openspec-new-change/SKILL.md
homonto/skills/openspec-verify-change/SKILL.md
```

- [ ] **Step 3: Copy required Superpowers skill directories**

Run:

```bash
for name in \
  brainstorming \
  writing-plans \
  executing-plans \
  subagent-driven-development \
  using-git-worktrees \
  test-driven-development \
  systematic-debugging \
  verification-before-completion \
  finishing-a-development-branch \
  requesting-code-review \
  receiving-code-review \
  dispatching-parallel-agents; do
  test -d "/home/mg/.agents/skills/$name"
  rm -rf "homonto/skills/$name"
  cp -a "/home/mg/.agents/skills/$name" "homonto/skills/$name"
done
```

Expected: no output and every copied directory has a `SKILL.md`.

- [ ] **Step 4: Replace dogfood skill declarations in `homonto.toml`**

Replace the current `[skills.onto*]` declarations with project-scoped local
Comet/OpenSpec/Superpowers declarations. Keep existing `[models.*]` tables.

Use this skill block:

```toml
[skills.comet]
source = "local:comet"
scope = "project"

[skills.comet-open]
source = "local:comet-open"
scope = "project"

[skills.comet-design]
source = "local:comet-design"
scope = "project"

[skills.comet-build]
source = "local:comet-build"
scope = "project"

[skills.comet-verify]
source = "local:comet-verify"
scope = "project"

[skills.comet-archive]
source = "local:comet-archive"
scope = "project"

[skills.comet-hotfix]
source = "local:comet-hotfix"
scope = "project"

[skills.comet-tweak]
source = "local:comet-tweak"
scope = "project"

[skills.openspec-explore]
source = "local:openspec-explore"
scope = "project"

[skills.openspec-new-change]
source = "local:openspec-new-change"
scope = "project"

[skills.openspec-verify-change]
source = "local:openspec-verify-change"
scope = "project"

[skills.brainstorming]
source = "local:brainstorming"
scope = "project"

[skills.writing-plans]
source = "local:writing-plans"
scope = "project"

[skills.executing-plans]
source = "local:executing-plans"
scope = "project"

[skills.subagent-driven-development]
source = "local:subagent-driven-development"
scope = "project"

[skills.using-git-worktrees]
source = "local:using-git-worktrees"
scope = "project"

[skills.test-driven-development]
source = "local:test-driven-development"
scope = "project"

[skills.systematic-debugging]
source = "local:systematic-debugging"
scope = "project"

[skills.verification-before-completion]
source = "local:verification-before-completion"
scope = "project"

[skills.finishing-a-development-branch]
source = "local:finishing-a-development-branch"
scope = "project"

[skills.requesting-code-review]
source = "local:requesting-code-review"
scope = "project"

[skills.receiving-code-review]
source = "local:receiving-code-review"
scope = "project"

[skills.dispatching-parallel-agents]
source = "local:dispatching-parallel-agents"
scope = "project"
```

- [ ] **Step 5: Verify config parses and projects**

Run:

```bash
go run . plan
```

Expected: plan shows creation or relocation of Comet/OpenSpec/Superpowers skill
links and deletion/pruning of old managed Onto links. It must not show writes to
tool JSON files unless unrelated managed config changed.

- [ ] **Step 6: Apply dogfood projection**

Run:

```bash
go run . apply --yes
go run . status
go run . doctor
```

Expected:

```text
No drift.
```

`doctor` may warn that `pass` is missing, but every declared skill link must be
reported as linked for both Claude and OpenCode.

- [ ] **Step 7: Commit local Comet projection**

Run:

```bash
git add homonto.toml homonto/skills
git commit -m "chore: dogfood comet development skills"
```

---

### Task 3: Update Living Development Workflow Documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/NEXT_AGENT.md`
- Modify: `docs/guides/README.md`
- Create: `docs/guides/comet-workflow.md`
- Modify: `docs/changes/README.md`
- Modify: `docs/guides/onto-workflow.md`

**Interfaces:**
- Consumes: Design doc `docs/superpowers/specs/2026-07-09-comet-development-workflow-migration-design.md`.
- Produces: Living docs that route future development through `/comet` and mark Onto docs as legacy/product-scoped.

- [ ] **Step 1: Update README contributor workflow**

In `README.md`, replace the contributor section that says this repo is developed
with `onto` with this content:

```markdown
### Development workflow

This repo is developed with **Comet**: OpenSpec owns WHAT, Superpowers owns HOW,
and Comet state/scripts bind the phases together. New development starts with
`/comet`; active changes live under `openspec/changes/`, and deep technical
designs and implementation plans live under `docs/superpowers/`.

The older `docs/changes/` Onto workspaces are historical. Do not open new work
there.

Future agents should start with [docs/NEXT_AGENT.md](docs/NEXT_AGENT.md) before
trusting older reviews or archived change artifacts.
```

- [ ] **Step 2: Update `docs/NEXT_AGENT.md` first-stop instructions**

Add a current workflow block near the top:

```markdown
## Current Development Workflow

Use Comet for new work. On entry:

1. Run `openspec list --json --no-color` to discover active OpenSpec changes.
2. Inspect `openspec/changes/<name>/.comet.yaml` for phase/state when a change exists.
3. Route through `/comet`; do not create new `docs/changes/*` Onto workspaces.
4. Treat `docs/changes/archive/*` as historical evidence only.
```

Update the recommended next steps so the next product work is opened as a Comet
change, with framework/catalog projection as the likely first product change.

- [ ] **Step 3: Update guide index**

In `docs/guides/README.md`, add `comet-workflow.md` as the active development
workflow guide and mark `onto-workflow.md` as legacy/internal history or product
framework reference.

- [ ] **Step 4: Create `docs/guides/comet-workflow.md`**

Create this file:

```markdown
# The Comet Development Workflow

Comet is Homonto's development workflow. OpenSpec owns WHAT: proposals,
requirements, delta specs, and archive semantics. Superpowers owns HOW: deep
technical design, implementation plans, execution discipline, verification, and
branch finishing. Comet state and scripts bind the two.

## Quick Start

- New work: `/comet <what you want to build>`
- Resume work: `/comet`
- Bug fix: `/comet-hotfix <symptom>` when it is an existing behavior bug
- Small tweak: `/comet-tweak <change>` when it is copy/config/docs/prompt-scale

## Layout

```text
.comet/config.yaml
openspec/changes/<name>/.comet.yaml
openspec/changes/<name>/{proposal.md,design.md,tasks.md}
openspec/specs/<capability>/spec.md
docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md
docs/superpowers/plans/YYYY-MM-DD-<feature>.md
docs/superpowers/reports/YYYY-MM-DD-<change>-verify.md
```

## Phase Model

1. Open: clarify goals, non-goals, scope, scenarios, and create OpenSpec artifacts.
2. Design: use Superpowers brainstorming to produce the deep technical design doc.
3. Build: write an implementation plan, choose isolation/execution/TDD/review mode, then execute.
4. Verify: run evidence-based verification and finish branch handling.
5. Archive: merge OpenSpec delta specs into main specs and archive the change.

## Gates

Comet has blocking user decisions for requirements confirmation, change name,
design approach, plan-ready workflow configuration, verify failures, branch
handling, and archive confirmation. Agents must not infer these choices from
history or defaults.

## Legacy Onto Artifacts

`docs/changes/` and `docs/changes/archive/` are legacy Onto history. They are
useful context, but new work must use `openspec/changes/`.
```

- [ ] **Step 5: Mark `docs/changes/README.md` legacy**

Add this paragraph at the top:

```markdown
> Legacy note: this directory documents the previous Onto workflow. New Homonto
> development uses Comet and OpenSpec under `openspec/changes/`. Keep archived
> Onto workspaces as historical evidence; do not create new active workspaces in
> `docs/changes/`.
```

- [ ] **Step 6: Mark `docs/guides/onto-workflow.md` non-current**

Add this paragraph after the title:

```markdown
> Legacy/internal note: Homonto development now uses Comet. This guide documents
> the previous Onto workflow and may still inform product-framework work if Onto
> remains a bundled user framework, but it is not the current repo development
> workflow.
```

- [ ] **Step 7: Commit documentation routing changes**

Run:

```bash
git add README.md docs/NEXT_AGENT.md docs/guides/README.md docs/guides/comet-workflow.md docs/changes/README.md docs/guides/onto-workflow.md
git commit -m "docs: route development workflow through comet"
```

---

### Task 4: Add Living Comet Workflow Spec And Preserve Product Boundaries

**Files:**
- Create: `docs/specs/comet-workflow.md`
- Modify: `docs/specs/onto-workflow.md`
- Modify: `docs/road-to-release.md`
- Modify: `docs/roadmap.md`
- Modify: `docs/release-checklist.md`
- Modify: `docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`

**Interfaces:**
- Consumes: Current dual-binary release design and Comet migration design.
- Produces: Living specs that distinguish internal development workflow from product framework commitments.

- [ ] **Step 1: Create `docs/specs/comet-workflow.md`**

Create this file:

```markdown
# comet-workflow Specification

## Purpose

Defines Homonto's current development workflow: Comet coordinates OpenSpec WHAT
artifacts with Superpowers HOW artifacts, state, verification, and archive.

## Requirements

### Requirement: Comet is the development entry point

New Homonto development SHALL start through `/comet` or a Comet preset. Agents
SHALL inspect `openspec/changes/` and each active change's `.comet.yaml` before
starting or resuming work. Agents SHALL NOT create new active `docs/changes/*`
Onto workspaces for Homonto development.

#### Scenario: No active change

- **GIVEN** `openspec list --json --no-color` returns no active changes
- **WHEN** the user requests new development work
- **THEN** the agent routes through `/comet-open` to create an OpenSpec change

### Requirement: OpenSpec is canonical for WHAT

New requirement changes SHALL be represented as OpenSpec changes under
`openspec/changes/<name>/`, with main specs under `openspec/specs/` after archive.
Existing `docs/specs/*.md` remain readable transition documents until a separate
conversion change migrates them.

#### Scenario: New capability

- **GIVEN** a new capability request
- **WHEN** Comet opens the change
- **THEN** proposal/design/tasks and any delta specs are created under
  `openspec/changes/<name>/`

### Requirement: Superpowers remains canonical for HOW

Deep technical design docs SHALL live under `docs/superpowers/specs/`, plans
under `docs/superpowers/plans/`, and verification reports under
`docs/superpowers/reports/`.

#### Scenario: Build phase planning

- **GIVEN** a Comet change in build phase
- **WHEN** the implementation plan is written
- **THEN** it is saved under `docs/superpowers/plans/` and its frontmatter links
  back to the OpenSpec change

### Requirement: Onto artifacts are legacy for development

`docs/changes/` SHALL be treated as legacy Onto history for Homonto development.
Archived workspaces MAY be consulted for historical context but SHALL NOT be
edited or used as active workflow state.

#### Scenario: Archived Onto change

- **GIVEN** an archived workspace under `docs/changes/archive/`
- **WHEN** an agent needs historical context
- **THEN** it may read the archive but must use current living docs and OpenSpec
  state for new work
```

- [ ] **Step 2: Mark `docs/specs/onto-workflow.md` non-current**

Add this paragraph after the purpose:

```markdown
> Legacy/internal note: this spec describes the previous Onto workflow. Homonto
> development now uses the Comet workflow described in `comet-workflow.md`. Keep
> this spec only as historical context or product-framework reference until an
> explicit product decision removes or rewrites Onto.
```

- [ ] **Step 3: Update release/product docs with boundary language**

In `docs/road-to-release.md`, `docs/roadmap.md`, `docs/release-checklist.md`, and
`docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`, make only these
semantic changes:

- Replace claims that "this repo is developed with Onto" with "this repo is
  developed with Comet".
- Preserve product-release statements about `onto` unless the text specifically
  talks about internal repo development.
- Add one sentence where needed: "Internal development workflow and bundled user
  framework scope are separate decisions."

- [ ] **Step 4: Run stale workflow grep**

Run:

```bash
rg 'Start with `/onto`|developed with \*\*onto\*\*|New work: `/onto' \
  README.md docs/NEXT_AGENT.md docs/guides/README.md docs/guides/comet-workflow.md \
  docs/road-to-release.md docs/roadmap.md docs/release-checklist.md
```

Expected: no living-doc result says current Homonto development starts with Onto.
Legacy Onto docs and archived change directories are intentionally excluded from
this current-entry check.

- [ ] **Step 5: Commit workflow spec changes**

Run:

```bash
git add docs/specs/comet-workflow.md docs/specs/onto-workflow.md docs/road-to-release.md docs/roadmap.md docs/release-checklist.md docs/superpowers/specs/2026-07-09-dual-binary-release-design.md
git commit -m "docs: specify comet as development workflow"
```

---

### Task 5: Verify Bootstrap And Create A Disposable Comet Smoke Change

**Files:**
- Temporary only: `openspec/changes/comet-bootstrap-smoke/` during the task.
- Modify only if needed: `.gitignore` if OpenSpec/Comet generated cache files appear and should stay untracked.

**Interfaces:**
- Consumes: Tasks 1-4 complete.
- Produces: Evidence that OpenSpec and Comet are operational without leaving a stale active smoke change.

- [ ] **Step 1: Run Homonto verification**

Run:

```bash
go run . apply --yes
go run . status
go run . doctor
```

Expected: `status` prints `No drift.` and `doctor` reports all declared Comet,
OpenSpec, and Superpowers skills linked for Claude and OpenCode. A missing `pass`
warning is acceptable.

- [ ] **Step 2: Create a disposable OpenSpec change**

Run:

```bash
openspec new change comet-bootstrap-smoke --description "Disposable smoke test for Comet bootstrap" --json --no-color
```

Expected: output JSON names `comet-bootstrap-smoke`, and directory
`openspec/changes/comet-bootstrap-smoke/` exists.

- [ ] **Step 3: Initialize Comet state for the smoke change**

Run:

```bash
COMET_ENV="${COMET_ENV:-$(find . "$HOME"/.*/skills "$HOME/.config" "$HOME/.gemini" -path '*/comet/scripts/comet-env.mjs' -type f -print -quit 2>/dev/null)}"
COMET_SCRIPTS_DIR="$(node "$COMET_ENV")"
COMET_STATE="$COMET_SCRIPTS_DIR/comet-state.mjs"
node "$COMET_STATE" init comet-bootstrap-smoke full
node "$COMET_STATE" check comet-bootstrap-smoke open
```

Expected: state check passes and
`openspec/changes/comet-bootstrap-smoke/.comet.yaml` exists.

- [ ] **Step 4: Remove the disposable smoke change**

Run:

```bash
rm -rf openspec/changes/comet-bootstrap-smoke
openspec list --json --no-color
```

Expected:

```json
{"changes":[]}
```

- [ ] **Step 5: Verify no stale smoke files remain**

Run:

```bash
test ! -e openspec/changes/comet-bootstrap-smoke
git status --short
```

Expected: no `comet-bootstrap-smoke` paths appear.

- [ ] **Step 6: Commit any ignore/config cleanup**

If Step 5 reveals only intended tracked files from previous tasks and no new
cleanup is needed, skip this commit. If `.gitignore` or another config file was
updated to keep generated caches out of git, run:

```bash
git add .gitignore
git commit -m "chore: ignore comet bootstrap cache files"
```

---

### Task 6: Final Regression And Handoff

**Files:**
- Modify: `docs/NEXT_AGENT.md` only if verification results need a final timestamp update.

**Interfaces:**
- Consumes: Tasks 1-5 complete.
- Produces: Verified clean `main` ready for the first real Comet-managed product change.

- [ ] **Step 1: Run full regression gate**

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./...
go run . status
go run . doctor
openspec list --json --no-color
```

Expected:

- Go tests pass.
- Vet reports no issues.
- Build succeeds.
- `go run . status` prints `No drift.`
- `doctor` reports all declared skills linked; missing `pass` warning is acceptable.
- `openspec list --json --no-color` prints `{"changes":[]}` unless a real Comet change has intentionally been opened.

- [ ] **Step 2: Run stale-doc checks**

Run:

```bash
rg 'Start with `/onto`|developed with \*\*onto\*\*|New work: `/onto' \
  README.md docs/NEXT_AGENT.md docs/guides/README.md docs/guides/comet-workflow.md \
  docs/road-to-release.md docs/roadmap.md docs/release-checklist.md
rg 'No OpenSpec changes directory found|OpenSpec changes directory found' \
  README.md docs/NEXT_AGENT.md docs/guides docs/road-to-release.md docs/roadmap.md \
  docs/release-checklist.md --glob '*.md' --glob '!docs/guides/onto-workflow.md'
```

Expected: no living doc directs current development to Onto. Historical or legacy-marked hits are acceptable only when the surrounding text says they are legacy.

- [ ] **Step 3: Update handoff with verification evidence if needed**

If `docs/NEXT_AGENT.md` does not already include the Comet migration verification
evidence, add a short block:

```markdown
Latest Comet migration checks on 2026-07-09:

- `openspec list --json --no-color` works.
- `go test ./... -count=1` passed.
- `go vet ./...` passed.
- `go build ./...` succeeded.
- `go run . status` reported `No drift.`
```

- [ ] **Step 4: Commit final handoff update if changed**

Run only if Step 3 modified `docs/NEXT_AGENT.md`:

```bash
git add docs/NEXT_AGENT.md
git commit -m "docs: record comet migration verification"
```

- [ ] **Step 5: Final status**

Run:

```bash
git status --short --branch
git log --oneline -5
```

Expected: clean worktree on `main`. Next product work should start with `/comet`, likely for `framework-catalog-projection`.
