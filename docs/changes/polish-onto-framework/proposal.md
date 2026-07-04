# Proposal: polish-onto-framework

## Why

onto v1 shipped working but carries known gaps that keep it short of
state-of-the-art: the `execution: subagent` mode is named in the state
schema but has **no defined protocol**; artifacts are described in prose
with **no canonical templates** (drift between changes is guaranteed); the
open/design conversation before artifacts exist is the one
**compaction-vulnerable** stretch with no checkpoint; close **merges specs
with no format validation** (the retired CLI used to catch SHALL-placement
and structure errors — onto currently would not); and the workflow has no
multi-agent leverage despite fresh evidence (the v1 dry-run agents found 11
real defects) that adversarial agents are its best quality tool.

## What Changes

Seven axes, all user-selected (2026-07-04):

1. **Multi-agent orchestration** — define the subagent build protocol
   (per-task fresh-context implementer + reviewer), adversarial multi-agent
   verification (independent skeptics attempt to refute each scenario), and
   optional parallel approach exploration in design.
2. **Artifact templates** — canonical templates for state.yaml, proposal,
   design, tasks, plan, verification, and ADR drafts, shipped as reference
   files bundled with the skills.
3. **Context-loss checkpoints** — incremental `notes.md` checkpoint during
   open/design clarification, plus explicit per-skill compaction-recovery
   instructions.
4. **Close-phase validation** — format lint at close: SHALL placement,
   scenario structure, RENAMED merge semantics, dangling-reference audit.
5. **Parallel-change coordination** — `deps:` in state.yaml, dispatcher
   awareness of blocked/stacked changes, worktree-per-change guidance.
6. **Ship handoff** — optional post-close contract handing the archived
   change's verification evidence to PR-creation skills as the PR body.
7. **Metrics in archive** — close stamps phase durations, task count,
   verify rounds, and upgrade events into the archived state.yaml.

## Capability Impact

- **Modified**: `onto-workflow` (docs/specs/onto-workflow.md) — most
  requirements gain or change behavior; delta spec required.
- Untouched: all homonto binary capabilities (apply-pipeline, cli-commands,
  config-model, secret-references, tool-adapters). **No Go source changes
  planned**; if dogfooding exposes a product bug again, scope amendment
  requires user confirmation (per v1 precedent).

## Not split

All seven axes modify the same eight SKILL.md files and the same layout
contracts; templates feed checkpoints feed orchestration, and validation/
metrics/ship all land in onto-close. Splitting would produce serial changes
editing identical files with heavy conflict surface and no independent
deliverability — kept as one change (recorded per the split-preflight rule).

## Grounding

graphify index built over the repo (user-approved) before design; design
claims must cite graph queries or direct file reads of the skill sources.

## Impact

- Modified: all 8 `content/skills/onto*/SKILL.md`, `docs/changes/README.md`,
  `docs/specs/README.md`, `docs/adr/README.md`, `docs/guides/README.md`,
  `docs/guides/onto-workflow.md`.
- New: template reference files (location decided in design: skill-bundled
  `references/` vs `docs/` tree), delta spec `specs/onto-workflow.md`.
- The live symlinks mean every skill edit is instantly active — validation
  dry-runs must gate on committed state, not works-in-progress.
