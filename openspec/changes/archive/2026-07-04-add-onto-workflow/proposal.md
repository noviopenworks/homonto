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

- `tool-adapters`: bug fix discovered during dogfooding — owned-skill
  symlinks are created only inside adapter `Apply`, are absent from the
  ChangeSet, and `apply` short-circuits on an empty plan, so a skills-only
  config never links anything (violates the existing "Idempotent link
  creation" scenario). Fix: pending links become first-class plan changes
  in both adapters. (Scope amendment confirmed by user 2026-07-04;
  supersedes the original "no Go source changes" non-goal.)

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
