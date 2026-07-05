# The onto Development Workflow

**onto** is this repo's development workflow: five phases — open → design →
build → verify → close — plus two preset fast paths (`/onto-fix` for bugs,
`/onto-tweak` for small non-bug changes). It is self-contained: eight
markdown skills (shipped from `content/skills/` in this very repo and
symlinked into your tools by `homonto apply`), one `docs/` tree for all
artifacts, and an agent-managed `state.yaml` per change. No workflow CLIs,
no scripts.

Every artifact has a **canonical template** bundled with the skill that
creates it (`content/skills/<skill>/references/`) — skills stay lean
process prose; payload loads only when a phase needs it, and structural
deviation from a template is a close-phase lint finding.

## Quick start

- New work: `/onto <what you want to build>`
- Bug: `/onto-fix <symptom>` · Small tweak: `/onto-tweak <what>`
- Resume anything (including after context loss): just `/onto`

The dispatcher always runs the same four steps: **preflight** (rtk +
graphify recommended — warns and proceeds when missing, see Recommended
tooling), **discovery** (find
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
  `notes.md`, `proposal.md`, `tasks.md`) from templates. Gates:
  clarification-complete, artifact review.
- **design** (`onto-design`) — ground-truth exploration, 2–3 approaches
  (optionally sketched by parallel agents when genuinely open), user
  confirms one; then `design.md`, ADR drafts (unnumbered, Proposed), and
  delta specs with testable scenarios. Gate: approach confirmation. No
  implementation code in this phase, ever.
- **build** (`onto-build`) — `plan.md` with bite-sized verified tasks;
  plan-ready gate (isolation / execution / tdd recorded in `state.yaml`);
  one commit per task; root-cause-first debugging on any failure. With
  `execution: subagent`, a coordinator dispatches one fresh implementer
  agent per task and fault-finding reviewers after risky tasks
  (protocol: `onto-build/references/subagent-protocol.md`).
- **verify** (`onto-verify`) — scale-appropriate check of every delta-spec
  scenario with fresh command output as evidence, then an **adversarial
  pass**: in full mode two fresh-context skeptics (conformance +
  robustness) try to refute the claims → `verification.md`. Gate on
  failure: fix or accept-deviation.
- **close** (`onto-close`) — **lint** the change (delta format, workspace
  state, dangling references — findings block), merge delta specs into
  `docs/specs/` (incl. RENAMED), number + accept ADRs into `docs/adr/`,
  satisfy the guides obligation, finalize metrics, final confirmation,
  archive the workspace, then offer a ready-made **ship handoff** (PR body
  from the archived evidence) for the PR skills.

## Checkpoints and recovery

Two complementary recovery mechanisms survive context loss: the
**phase-derivation table** (where the change is — recomputed from files on
every dispatch) and **notes.md** (why — confirmed facts, pending items,
grounding; updated before ending any decision-producing turn in
open/design). After a compaction, skills read notes.md first and resume
from its Pending items instead of re-asking answered questions.

## Parallel changes

`state.yaml` `deps:` names changes that must archive first; the dispatcher
shows deps status and warns before resuming a blocked change. For several
simultaneously active changes, use one git worktree per change. Metrics
(`metrics:` in state.yaml — phase dates, task count, verify rounds,
upgrades) are stamped along the way, purely observational.

## Presets and upgrade rules

- `/onto-fix` — broken behavior. Failing test reproducing the bug comes
  first, always. Upgrades to full workflow (with design backfill) on: 3+
  files, architecture/schema changes, new public API, or scope beyond one
  function/module.
- `/onto-tweak` — copy/config/docs/prompt changes, plus small features
  within tweak limits: ≤5 files (tests excluded), no new capability, no
  existing-spec requirement change. Upgrades on: 5+ files, cross-module
  coordination, 5+ new tests, config key add/remove, a new capability, or
  spec-affecting changes.
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

## Recommended tooling

onto recommends two tools; when either is missing the dispatcher warns and
proceeds — a degraded session still works:

- **rtk** — token-optimized CLI proxy; workflow shell operations go through
  it when installed. Missing rtk means higher token costs, never a stop.
- **graphify** (https://graphify.net) — codebase understanding; the open and
  design phases ground claims in graphify/codegraph queries when available.
  Without the skill and without an existing index, grounding falls back to
  direct file reading and the fallback is recorded in the change's notes.

## This repo eats it first

The skills live in `content/skills/onto*` and are listed in `homonto.toml`
under `[skills] own`, so `homonto apply` links them into `~/.claude/skills/`
(and OpenCode). Editing a skill file is instantly live everywhere — that is
homonto's owned-content model doing its job.
