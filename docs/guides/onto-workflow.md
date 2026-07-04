# The onto Development Workflow

**onto** is this repo's development workflow: five phases — open → design →
build → verify → close — plus two preset fast paths (`/onto-fix` for bugs,
`/onto-tweak` for small non-bug changes). It is self-contained: eight
markdown skills (shipped from `content/skills/` in this very repo and
symlinked into your tools by `homonto apply`), one `docs/` tree for all
artifacts, and an agent-managed `state.yaml` per change. No workflow CLIs,
no scripts.

## Quick start

- New work: `/onto <what you want to build>`
- Bug: `/onto-fix <symptom>` · Small tweak: `/onto-tweak <what>`
- Resume anything (including after context loss): just `/onto`

The dispatcher always runs the same four steps: **preflight** (rtk +
graphify must be installed — see Required tooling), **discovery** (find
active changes under `docs/changes/`), **derivation** (compute the real
phase from files, correcting `state.yaml` if it drifted), and **routing**
(load the matching phase skill).

## The layout

```
docs/
├── adr/                    # accepted decisions — docs/adr/README.md
├── specs/                  # living capability specs — docs/specs/README.md
├── changes/                # change workspaces + archive — docs/changes/README.md
│   ├── <name>/             # active change (state.yaml, proposal, design, …)
│   └── archive/YYYY-MM-DD-<name>/
└── guides/                 # user-facing docs — docs/guides/README.md
```

## Phase walkthrough

- **open** (`onto-open`) — clarify until unambiguous, check whether the work
  should split into multiple changes, create the workspace (`state.yaml`,
  `proposal.md`, `tasks.md`). Gates: clarification-complete, artifact review.
- **design** (`onto-design`) — ground-truth exploration, 2–3 approaches,
  user confirms one; then `design.md`, ADR drafts (unnumbered, Proposed),
  and delta specs with testable scenarios. Gate: approach confirmation. No
  implementation code in this phase, ever.
- **build** (`onto-build`) — `plan.md` with bite-sized verified tasks;
  plan-ready gate (isolation / execution / tdd recorded in `state.yaml`);
  one commit per task; root-cause-first debugging on any failure.
- **verify** (`onto-verify`) — scale-appropriate check of every delta-spec
  scenario with fresh command output as evidence → `verification.md`.
  Gate on failure: fix or accept-deviation.
- **close** (`onto-close`) — merge delta specs into `docs/specs/`, number +
  accept ADRs into `docs/adr/`, satisfy the guides obligation (update
  `docs/guides/` or record a waiver), final confirmation, archive the
  workspace.

## Presets and upgrade rules

- `/onto-fix` — broken behavior. Failing test reproducing the bug comes
  first, always. Upgrades to full workflow (with design backfill) on: 3+
  files, architecture/schema changes, new public API, or scope beyond one
  function/module.
- `/onto-tweak` — copy/config/docs/prompt changes. Upgrades on: 5+ files,
  cross-module coordination, 5+ new tests, config key add/remove, a new
  capability, or spec-affecting changes.
- When in doubt, start full (`/onto`): presets exist for speed, not for
  dodging design.

## GitHub entry points

- **resolve-issue** → entry into onto: the issue seeds `onto-open`
  clarification (fix preset for bugs, full workflow for features), usually
  in a worktree.
- **continue-pr** → entry into onto: PR review feedback resumes the matching
  change's build phase, or opens a fix change referencing the PR.
- PR creation and PR review are separate skills, not onto phases — onto ends
  at a verified, closed change on a branch.

## Required tooling

onto hard-requires two tools and halts with install instructions when
either is missing:

- **rtk** — token-optimized CLI proxy; all workflow shell operations go
  through it.
- **graphify** (https://graphify.net) — codebase understanding; the open and
  design phases must ground claims in graphify/codegraph queries instead of
  guesswork.

## This repo eats it first

The skills live in `content/skills/onto*` and are listed in `homonto.toml`
under `[skills] own`, so `homonto apply` links them into `~/.claude/skills/`
(and OpenCode). Editing a skill file is instantly live everywhere — that is
homonto's owned-content model doing its job.
