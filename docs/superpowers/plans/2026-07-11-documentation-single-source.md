# Documentation Single Source of Truth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `docs/roadmap.md` the standalone project truth, migrate unique requirements into OpenSpec, move completed Superpowers evidence into OpenSpec archives, and delete stale, duplicate, and superseded documentation.

**Architecture:** Documentation-only migration in dependency-ordered stages. Each stage ends with a stale-phrase scan, link check, and the existing Go/Docker gates so a broken reference never lands on `main`. No product behavior changes.

**Tech Stack:** Markdown, YAML (`.comet.yaml`), shell, Go (only for the build/smoke gate), Git.

## Global Constraints

- This plan touches documentation and archive metadata only. Do not edit Go source, tests, CI workflows, or scripts in this plan.
- Never discard a concurrent uncommitted edit to `docs/roadmap.md` without first inspecting and attributing it; status claims without source/test evidence are not accepted.
- `openspec/changes/archive/*` and `docs/adr/*` are the only retained historical layers after this plan.
- `catalog/skills/` is the sole tracked bundled-skill source after this plan.
- Every Markdown link and every `.comet.yaml` path field must resolve after each stage.
- Checked roadmap boxes require direct evidence (test name, command, or file); the roadmap must not mark the agent prune deletion-failure fix complete until source and a focused regression test exist.
- Commit each stage separately so Git history remains a recovery path.

## File Structure

- Rewrite: `docs/roadmap.md` (standalone dashboard, all 11 fixed sections).
- Delete: `docs/road-to-release.md`, `docs/reviews/2026-07-04-deep-review.md`, `docs/specs/` (entire tree), `docs/changes/` (entire tree), `docs/superpowers/plans/2026-07-11-agentic-workflows-roadmap.md`, all completed `docs/superpowers/{specs,plans,reports}/*`, duplicate `homonto/skills/*`.
- Modify: `README.md`, `docs/guides/using-homonto.md`, `docs/guides/onto-workflow.md`, `docs/guides/README.md`, `docs/adr/README.md`, six `openspec/specs/*/spec.md`, every archived `openspec/changes/archive/*/.comet.yaml` path field, `.gitignore`.
- Create: `docs/superpowers/README.md`, `openspec/specs/apply-pipeline/spec.md`, `openspec/specs/secret-references/spec.md`, `openspec/specs/cli-commands/spec.md`, `openspec/specs/comet-workflow/spec.md`, `homonto/skills/.gitkeep`.
- Move: each completed design/plan/report into its `openspec/changes/archive/<change>/` directory.

---

### Task 1: Rewrite the Standalone Roadmap

**Files:**
- Modify: `docs/roadmap.md` (full rewrite, 11 fixed sections)

**Interfaces:**
- Produces: the canonical status authority; every later task links subordinate docs to it.

- [ ] **Step 1: Inspect and attribute the concurrent worktree edit**

Run: `/usr/bin/git diff -- docs/roadmap.md`

The concurrent edit added "Implementation checks (verified 2026-07-11)" blocks and changed the test count to 413. Two of its `[x]` marks (prune deletion-failure retention, and de-declared-target record retention across update) claim safety work that source does not yet implement. Record the author/date, then proceed — the rewrite removes those unverified `[x]` marks and restores the 443 count with its verification command. Do not silently adopt unverified claims.

- [ ] **Step 2: Replace the entire file with the 11-section standalone dashboard**

Write `docs/roadmap.md` with exactly these top-level sections, in order:

1. `# Homonto Product and Engineering Roadmap` with an `Authority And Maintenance` subsection stating this file is the sole status/priority authority, the four-label status vocabulary (Implemented / Partial / Planned / Deferred), and that checked boxes require direct evidence.
2. `## Product Purpose` — one paragraph: Homonto is a declarative config projector for Claude Code and OpenCode with a plan/confirm/apply pipeline; onto is its sibling spec-driven workflow operator.
3. `## Architecture Summary` — homonto engine (config → adapters → atomic writes → state), onto CLI (ontostate phase machine), embedded catalog, agent lifecycle lockfile/blob/merge.
4. `## Implemented Capability Matrix` — two tables (homonto commands/capabilities, onto commands/capabilities) using the audit's definitive matrix from the design context. Mark `agents prune` as **Partial** with the deletion-failure defect named.
5. `## Partial And Unsafe Behavior` — the two agent ownership defects (de-declared target records dropped by `runAgentUpdate`; `pruneFile` ignores `os.Remove` errors), the Docker-image-builds-only-homonto gap, and the release-workflow-weak-gate gap.
6. `## Not Implemented` — remote sources, third adapter, interactive TUI, OpenCode comment preservation, broad import, blob GC, per-agent scope.
7. `## Current Release Gate` — the four release-integrity items (agent ownership safety, documentation truth, dual-binary Docker E2E, unified release gate) each marked open with its exit gate.
8. `## Implementation Backlog` — the 11 dependency-ordered work packages from the design, each with problem, scope, dependencies, primary files, acceptance scenarios, verification command, and exit gate. Mark only package items with real evidence as done.
9. `## Twelve-Month Direction` — stabilization, resource coherence, remote trust, ecosystem expansion, each with its exit gate.
10. `## Documentation And Archive Map` — the authority hierarchy: roadmap (status), README (install), openspec/specs (behavior), docs/adr (decisions), docs/guides (usage), docs/release-checklist (mechanics), openspec/changes/archive (history).
11. `## Verified Evidence Ledger` — the exact commands and date: `go test ./... -count=1` → 443 passed in 26 packages (2026-07-11); `go test -race ./...` → 443 passed; `go vet ./...` clean; `./scripts/docker-test.sh` → `SMOKE PASS`; `git tag --list` → none.

Do not carry over the implementation-diary narrative from the old roadmap. Each matrix row is one line. Each backlog entry links to its evidence files.

- [ ] **Step 3: Verify no unverified checked boxes remain**

Run: `rg -n '\[x\]' docs/roadmap.md`

Expected: every `[x]` line cites a test name, a command, or a file path in the same entry. The two agent-ownership `[x]` marks from the concurrent edit must be gone.

- [ ] **Step 4: Verify the test-count claim names its command**

Run: `rg -n '443|413' docs/roadmap.md`

Expected: the ledger names `go test ./... -count=1` next to the 443 figure; no bare 413 figure remains as a current baseline.

- [ ] **Step 5: Commit the standalone roadmap**

```bash
git add docs/roadmap.md
git commit -m "docs: make roadmap the standalone project truth"
```

### Task 2: Migrate Transitional Specs Into OpenSpec

**Files:**
- Create: `openspec/specs/apply-pipeline/spec.md`
- Create: `openspec/specs/secret-references/spec.md`
- Create: `openspec/specs/cli-commands/spec.md`
- Create: `openspec/specs/comet-workflow/spec.md`
- Modify: `openspec/specs/config-model/spec.md`
- Modify: `openspec/specs/tool-adapters/spec.md`
- Delete: entire `docs/specs/` tree

**Interfaces:**
- Produces: OpenSpec as the sole capability-spec tree; `docs/specs/` removed.

- [ ] **Step 1: Create the four missing OpenSpec capabilities**

For each of `apply-pipeline`, `secret-references`, `cli-commands`, `comet-workflow`: create `openspec/specs/<capability>/spec.md` whose `## Purpose` is one paragraph and whose `## Requirements` are copied from the corresponding `docs/specs/<capability>.md`, with these corrections:

- `cli-commands`: add the full agent command surface (`agents list`, `add`, `update [--all]`, `doctor`, `prune`) and correct `doctor` to cover expanded skills, commands, and subagents for both tools. Remove the statement that builtin lookup is unimplemented.
- `comet-workflow`: state that OpenSpec main specs are canonical and `docs/specs/` is removed.
- `apply-pipeline` and `secret-references`: copy verbatim; their content is current.

- [ ] **Step 2: Correct stale claims in existing OpenSpec specs**

In `openspec/specs/config-model/spec.md`: add the builtin-agent effective-mode exception (builtin with omitted mode defaults to `copy`; builtin with explicit `link` is rejected) next to the general `link` default.

In `openspec/specs/tool-adapters/spec.md`: delete the sentence at the Claude projection requirement that says builtin catalog lookup is not implemented; state that builtin resources resolve from the versioned materialized catalog.

- [ ] **Step 3: Replace the six generated TBD Purpose sections**

For `agent-lifecycle`, `subagent-projection`, `framework-expansion`, `builtin-catalog`, `command-projection`, and `onto-binary`: replace each `TBD - created by archiving change …` line with a one-paragraph capability purpose.

- [ ] **Step 4: Verify no TBD purposes remain**

Run: `rg -n 'TBD - created by archiving' openspec/specs`

Expected: no output.

- [ ] **Step 5: Verify the unique requirements are now in OpenSpec**

Run: `rg -n '### Requirement:' openspec/specs/apply-pipeline openspec/specs/secret-references openspec/specs/cli-commands openspec/specs/comet-workflow`

Expected: each new spec has at least one requirement heading.

- [ ] **Step 6: Delete the transitional spec tree**

```bash
git rm -r docs/specs
```

- [ ] **Step 7: Commit the spec migration**

```bash
git add openspec/specs
git commit -m "docs: migrate transitional specs into OpenSpec"
```

### Task 3: Rewrite Guides And Correct README

**Files:**
- Modify: `README.md`
- Modify: `docs/guides/using-homonto.md`
- Modify: `docs/guides/onto-workflow.md`
- Modify: `docs/guides/README.md`
- Modify: `docs/adr/README.md`
- Delete: `docs/reviews/2026-07-04-deep-review.md`

**Interfaces:**
- Produces: guides describe current usage; README links to the roadmap for status; no guide declares implementation status.

- [ ] **Step 1: Correct the using-homonto guide**

In `docs/guides/using-homonto.md`:
- Replace lines 7-8 (only homonto CLI exists) with: both `homonto` and `onto` binaries build from source; `onto` is the spec-driven workflow operator.
- Replace lines 68-71 (obsolete plugin arrays) with the per-plugin table schema: `[plugins.claude.<name>] source=… enabled=… config={…}` and `[plugins.opencode.<name>] source=…`.
- Add a note after the config example that any config enabling a model tool must declare all three model routes (`architectural`, `coding`, `trivial`) for that tool, with a link to the roadmap's capability matrix.
- Remove any statement that there are no imperative mutators (the agent lifecycle has `add`/`update`/`prune`).

- [ ] **Step 2: Rewrite the onto-workflow guide as a product guide**

Rewrite `docs/guides/onto-workflow.md` as a guide to the implemented `onto` binary: `init`, `new`, `status`, `advance`, `close`, `doctor`, the phase order, the gates, and the archive layout. Remove all claims that onto is the repo's current development workflow or that it consists only of markdown skills with no CLI.

- [ ] **Step 3: Reduce the guides index**

Rewrite `docs/guides/README.md` to list the four guides with one-line descriptions and a single line: "Project status, release gate, and implementation backlog live in [`../roadmap.md`](../roadmap.md)." Remove the onto `state.yaml` guides-obligation procedure.

- [ ] **Step 4: Reduce the ADR index**

Rewrite `docs/adr/README.md` to describe ADRs as decision history (Accepted / Superseded) with the numbering rule. Remove the `docs/changes/` staging procedure and the `onto-close` numbering instruction; state that new ADRs are staged inside an active OpenSpec change and numbered at archive.

- [ ] **Step 5: Correct README**

In `README.md`: remove any `road-to-release.md` link and replace it with a link to `docs/roadmap.md`. Ensure the "For contributors" section points at `docs/roadmap.md` for current status and `docs/guides/comet-workflow.md` for the workflow. Leave install/quickstart/config/limitations sections intact.

- [ ] **Step 6: Delete the superseded review**

```bash
git rm docs/reviews/2026-07-04-deep-review.md
```

- [ ] **Step 7: Verify no guide claims obsolete facts**

Run: `rg -n 'onto.*not implemented|plugin.*\[\]|no.*mutators|only.*homonto.*CLI' docs/guides README.md`

Expected: no output.

- [ ] **Step 8: Commit the guide and README corrections**

```bash
git add README.md docs/guides docs/adr/README.md
git commit -m "docs: correct guides, README, and ADR index"
```

### Task 4: Move Completed Superpowers Artifacts Into OpenSpec Archives

**Files:**
- Move: each `docs/superpowers/specs/<change>-design.md` → `openspec/changes/archive/<change>/technical-design.md`
- Move: each `docs/superpowers/plans/<change>.md` → `openspec/changes/archive/<change>/implementation-plan.md`
- Move: each `docs/superpowers/reports/<change>-verify.md` → `openspec/changes/archive/<change>/verification-report.md`
- Modify: every `openspec/changes/archive/*/.comet.yaml` path field
- Create: `docs/superpowers/README.md`

**Interfaces:**
- Produces: `docs/superpowers/` contains only the README plus active-work artifacts; every archive is self-contained.

- [ ] **Step 1: Build the move list from archive metadata**

For each directory under `openspec/changes/archive/`, read its `.comet.yaml` and extract the three path fields: `design_doc`, `plan`, `verification_report`. These map 1:1 to completed Superpowers artifacts. Confirm each source file exists.

- [ ] **Step 2: Move each artifact trio into its archive**

For each archived change, run:

```bash
git mv "<design_doc path>" "openspec/changes/archive/<change>/technical-design.md"
git mv "<plan path>" "openspec/changes/archive/<change>/implementation-plan.md"
git mv "<verification_report path>" "openspec/changes/archive/<change>/verification-report.md"
```

- [ ] **Step 3: Update every .comet.yaml path field**

In each `openspec/changes/archive/<change>/.comet.yaml`, rewrite the three fields to the new in-archive paths:

```yaml
design_doc: openspec/changes/archive/<change>/technical-design.md
plan: openspec/changes/archive/<change>/implementation-plan.md
verification_report: openspec/changes/archive/<change>/verification-report.md
```

- [ ] **Step 4: Handle the umbrella three-way-merge design**

`docs/superpowers/specs/2026-07-11-agents-3way-merge-design.md` has no single archive. Move it into `openspec/changes/archive/2026-07-11-agents-merge-update/technical-design.md` (the final implementing change) and add a `superseded-by` note at the top pointing to the merge-core/merge-update/update-all archive chain. If that path is occupied by the merge-update design from Step 2, instead move it to `openspec/changes/archive/2026-07-11-agents-merge-core/technical-design-umbrella.md` and reference it from the merge-core design.

- [ ] **Step 5: Handle the un-archived completed designs and plans**

These four completed designs/plans have no `archived-with:` metadata because they predate the Comet workflow or were implemented directly:

- `2026-07-09-comet-development-workflow-migration` (design + plan)
- `2026-07-09-dual-binary-release` (design)
- `2026-07-09-explicit-config-resource-model` (plan)
- `2026-07-10-project-development-instructions` (design + plan)

For each: confirm its content is represented by an ADR (0012 for comet migration; an existing release ADR or the roadmap for dual-binary; ADR 0011 / config-model spec for the resource model; `AGENTS.md` for project instructions). If represented, delete the file. If a unique decision is not represented, create an ADR first, then delete.

- [ ] **Step 6: Create the Superpowers README**

Create `docs/superpowers/README.md`:

```markdown
# Superpowers

Active implementation work only. Each design lives at `docs/superpowers/specs/`,
each plan at `docs/superpowers/plans/`, and each verification report at
`docs/superpowers/reports/` — but only while the corresponding OpenSpec change is
active.

When a change archives, its design, plan, and report move into
`openspec/changes/archive/<change>/` as `technical-design.md`,
`implementation-plan.md`, and `verification-report.md`. Nothing in this directory
is historical truth; it is work in progress.

Project status and the implementation backlog live in
[`../roadmap.md`](../roadmap.md).
```

- [ ] **Step 7: Verify no archive references the old paths**

Run: `rg -n 'docs/superpowers/(specs|plans|reports)' openspec/changes/archive`

Expected: no output (all `.comet.yaml` fields now point inside the archive).

- [ ] **Step 8: Verify the live Superpowers tree is empty of completed work**

Run: `rg -l 'archived-with:' docs/superpowers`

Expected: no output.

- [ ] **Step 9: Commit the archive consolidation**

```bash
git add openspec/changes/archive docs/superpowers
git commit -m "docs: consolidate superpowers history into OpenSpec archives"
```

### Task 5: Delete Legacy Onto Archives And Standalone Plan

**Files:**
- Delete: `docs/changes/` (entire tree including `archive/` and `README.md`)
- Delete: `docs/superpowers/plans/2026-07-11-agentic-workflows-roadmap.md`

**Interfaces:**
- Produces: the retired Onto workflow tree and the orphaned roadmap plan are gone; OpenSpec archives are the sole change history.

- [ ] **Step 1: Confirm no unique decision lives only in legacy Onto archives**

Run: `rg -n 'Status: Accepted|Decision:' docs/changes/archive | head`

For any accepted decision found, confirm it is represented in `docs/adr/`. If not, create an ADR before proceeding. The legacy archives are primarily retired-workflow state and proposals whose decisions are already in ADRs 0001-0012.

- [ ] **Step 2: Delete the legacy Onto tree**

```bash
git rm -r docs/changes
```

- [ ] **Step 3: Delete the standalone agentic workflows plan**

Its useful content was merged into the roadmap's Implementation Backlog in Task 1.

```bash
git rm docs/superpowers/plans/2026-07-11-agentic-workflows-roadmap.md
```

- [ ] **Step 4: Verify no living document references the deleted paths**

Run: `rg -n 'docs/changes/|2026-07-11-agentic-workflows-roadmap' README.md docs openspec/specs AGENTS.md`

Expected: no output.

- [ ] **Step 5: Commit the legacy cleanup**

```bash
git commit -m "docs: remove legacy onto archives and standalone roadmap plan"
```

### Task 6: Remove Duplicate Bundled Skill Content

**Files:**
- Delete: all files under `homonto/skills/` except `.gitkeep`
- Create: `homonto/skills/.gitkeep` (if not already present)

**Interfaces:**
- Produces: `catalog/skills/` is the sole tracked bundled-skill source; `homonto/skills/` is an empty local-provider directory.

- [ ] **Step 1: Verify the two trees are byte-identical**

Run: `diff -r catalog/skills homonto/skills && echo IDENTICAL`

Expected: `IDENTICAL`. If any difference appears, stop and reconcile before deleting — a divergence means one side has edits that must not be lost.

- [ ] **Step 2: Delete the duplicate content**

```bash
find homonto/skills -mindepth 1 ! -name '.gitkeep' -delete
```

Then ensure the local-provider directory remains tracked:

```bash
touch homonto/skills/.gitkeep
```

- [ ] **Step 3: Verify the build still embeds the catalog**

Run: `go build ./...`

Expected: success (the embed uses `catalog/skills/`, not `homonto/skills/`).

- [ ] **Step 4: Verify the dogfood state still materializes**

Run: `go run . status`

Expected: `No drift.` (the materialized catalog under `.homonto/` is regenerated from `catalog/skills/`).

- [ ] **Step 5: Commit the duplicate removal**

```bash
git add homonto/skills
git commit -m "docs: remove duplicate bundled skills under homonto/skills"
```

### Task 7: Add Ignore Policy And Final Verification

**Files:**
- Modify: `.gitignore`

**Interfaces:**
- Produces: scratch policy is visible at the repository root; the full consolidation gate passes.

- [ ] **Step 1: Add the root scratch ignore rule**

Append to `.gitignore`:

```
/.superpowers/
```

The nested `.superpowers/sdd/.gitignore` remains, but the root rule makes the policy visible from the top-level ignore file.

- [ ] **Step 2: Run the stale-phrase scan across all living docs**

Run:

```bash
rg -n 'not implemented yet|not yet merged|168/168|168 tests|foundation.*only|TBD - created|docs/NEXT_AGENT|road-to-release' README.md docs openspec/specs AGENTS.md
```

Expected: no output. Matches only count if they appear in living (non-archive) documents.

- [ ] **Step 3: Run a Markdown link check**

Run:

```bash
rg -no '\]\(([^)]+)\)' README.md docs/*.md docs/**/*.md openspec/specs/**/*.md | sed 's/.*](\(.*\))/\1/' | sort -u
```

Manually confirm each relative path resolves to an existing file. Fix any broken link.

- [ ] **Step 4: Run the Go gate**

Run:

```bash
gofmt -l .
go mod tidy -diff
go vet ./...
go build ./...
go test ./... -count=1
```

Expected: all clean; 443 or more tests pass.

- [ ] **Step 5: Run the Docker smoke**

Run: `./scripts/docker-test.sh`

Expected: `SMOKE PASS`.

- [ ] **Step 6: Verify the worktree has no generated caches staged**

Run: `/usr/bin/git status --short`

Expected: only the `.gitignore` change (and any Task 3-6 leftovers) are present; no `graphify-out/`, `.codegraph/`, `.homonto/`, or `.superpowers/` files are staged.

- [ ] **Step 7: Commit the ignore policy**

```bash
git add .gitignore
git commit -m "chore: surface superpowers scratch ignore at repo root"
```

## Self-Review Checklist

Run this before declaring the plan executed:

- [ ] `docs/roadmap.md` has all 11 sections and no unverified `[x]`.
- [ ] `docs/road-to-release.md`, `docs/specs/`, `docs/changes/`, and `docs/reviews/` are absent.
- [ ] `docs/superpowers/` contains only the README and any genuinely active artifacts.
- [ ] Every `openspec/changes/archive/*/.comet.yaml` points at in-archive paths.
- [ ] `catalog/skills/` is the sole bundled-skill source and `go build ./...` passes.
- [ ] No living document references `docs/NEXT_AGENT.md`, `road-to-release`, or `docs/changes/`.
- [ ] The stale-phrase scan and link check are clean.
- [ ] `go test ./... -count=1` and `./scripts/docker-test.sh` pass.
