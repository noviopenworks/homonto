---
name: onto
description: onto workflow dispatcher. Use when starting, resuming, or asking about any development work in a repo with the docs/ onto layout — runs tooling preflight, finds the active change, derives the real phase from file state, and routes to the matching onto sub-skill.
---

# onto — Workflow Dispatcher

onto is a self-contained, markdown-only development workflow. Five phases —
**open → design → build → verify → close** — plus two preset paths
(`onto-fix` for bugs, `onto-tweak` for small non-bug changes). All artifacts
live in one `docs/` tree; phase state lives in an agent-managed
`docs/changes/<name>/state.yaml` that is always cross-checked against real
file state. There are no scripts and no external workflow CLIs: the skills
are the machinery.

The dispatcher does exactly four things, in order: preflight → discover →
derive → route. It never performs phase work itself.

## 1. Tooling preflight (hard requirement — runs first, every dispatch)

Run these checks before anything else. If either fails, HALT the workflow
and print the install instructions — do not continue in a degraded mode.

1. **rtk** — run `rtk --version`. On success, all subsequent shell
   operations in every onto phase go through rtk (or the rtk hook rewrites
   them transparently). On failure, HALT and print:

   > onto requires **rtk** (token-optimized CLI proxy) and it was not found
   > on PATH. Install it, verify with `rtk --version`, then re-run `/onto`.

2. **graphify** — confirm codebase-understanding tooling is available: the
   `graphify` skill is loadable, or a `graphify-out/` directory or
   `.codegraph/` index exists at the repo root. The open and design phases
   MUST ground every codebase claim in graphify/codegraph queries rather
   than guesswork. Indexing is the user's decision: if only the skill is
   available and no index exists, ask the user whether to build one before
   open/design proceeds; if they decline, grounding falls back to direct
   file reading and that fallback is recorded in the proposal/design. On
   failure (neither skill nor index), HALT and print:

   > onto requires **graphify** (https://graphify.net) for codebase
   > understanding and neither the skill nor an existing index
   > (`graphify-out/`, `.codegraph/`) was found. Install/enable graphify,
   > then re-run `/onto`.

## 2. Active-change discovery

Scan `docs/changes/*/` excluding `archive/`. A change is active iff its
directory sits directly under `docs/changes/` and its `state.yaml` has
`archived: false` (or state.yaml is absent — it will be rebuilt, see below).

| Active changes | User input | Behavior |
|---|---|---|
| None | description given | Route to `onto-open` with the description |
| None | nothing | Ask what the user wants to work on, then `onto-open` |
| Exactly one | nothing | Resume it: derive phase, route |
| Exactly one | new description | ASK: continue the active change or open a new one |
| Two or more | anything | LIST them (name, workflow, claimed phase, deps status) and ASK which to resume before doing anything else |

**Dependencies**: each change's `state.yaml` may name `deps:` — changes
that must archive before this one builds. Discovery listings show deps
status (`ready` / `blocked by <name>`). Before resuming a change whose
deps are not all archived, warn and require an explicit user choice:
proceed anyway, switch to the dependency, or stop. For multiple
simultaneously active changes, recommend one git worktree per change —
coupled work that can't be separated should have been one change (the
split-preflight rule already says so).

If the repo has no `docs/changes/` tree at all, offer to bootstrap the
layout: create `docs/{adr,specs,changes/archive,guides}/` with their README
contracts (see the layout section of `docs/guides/onto-workflow.md` in a
repo that has them, or recreate from this skill set's contracts), then
proceed to `onto-open`.

## 3. Phase derivation and cross-check

`state.yaml` is a **cache of truth, not truth**. On every dispatch:

1. Read `state.yaml` (canonical schema, template, and per-field rebuild
   rules: `references/state-yaml.md` in this skill's directory; summary in
   `docs/changes/README.md`). If a skill's `references/` directory is ever
   missing, reconstruct from the `docs/` contract pointers, note the gap,
   and continue — degrade, never halt.
2. Independently derive the phase from artifacts with this table
   (**first match from the top wins — strongest evidence first**; it must
   stay identical to the copy in `docs/changes/README.md`):

| Evidence | Real phase |
|---|---|
| `archived: true` or workspace under `archive/` | done |
| `verification.md` with a `Result: pass` line | close |
| all tasks checked in `tasks.md` | verify |
| `design.md` marked `Status: Confirmed`, or a preset workspace | build |
| `proposal.md` + `tasks.md` exist (full workflow, no confirmed design) | design |
| workspace exists, artifacts incomplete | open |

3. **Files win downward; gates win upward.** If the derived phase is
   earlier than the claimed phase, correct `state.yaml` to match the files,
   tell the user what was corrected and why, and continue from the derived
   phase. If the derived phase is later than the claimed phase, do not
   silently promote — the phase field advances only when a phase's exit
   gate is answered, so a lagging claim means an unanswered gate: resume at
   the claimed phase's gate (artifacts already prepared) and let it advance
   normally.
4. A missing or malformed `state.yaml` is never an error: rebuild it per
   the per-field table in `references/state-yaml.md` (`workflow` from the
   proposal's `Preset:` marker, else the branch prefix, else `full`;
   `base_ref` = parent of the oldest commit touching the workspace;
   `decisions` reset to null so gates are re-asked; `verify.result` from
   verification.md's `Result:` line; `deps` from the proposal's
   `Depends-on:` line; `metrics` best-effort, never blocking), announce
   the rebuild, continue.
5. Never trust conversation history for phase detection — after context
   loss or compaction, this derivation is the recovery mechanism. Re-run it.

## 4. Routing table

| Derived state | Load skill |
|---|---|
| `workflow: fix` (any phase) | `onto-fix` — presets own their whole lifecycle |
| `workflow: tweak` (any phase) | `onto-tweak` — presets own their whole lifecycle |
| phase open | `onto-open` |
| phase design | `onto-design` |
| phase build | `onto-build` |
| phase verify | `onto-verify` |
| phase close | `onto-close` |
| done | Report that the change is archived; ask what's next |

New work routes by intent: bug fix with clear reproduction → `onto-fix`;
copy/config/docs/prompt touch-up → `onto-tweak`; anything needing design →
`onto-open` (full). Preset skills contain upgrade rules that force the full
path when scope grows — never talk a change *down* from full to a preset.

## 5. GitHub entry points (contract)

- **Issue intake** (e.g. a resolve-issue skill): the issue text seeds
  `onto-open` clarification — fix preset for bugs, full workflow for
  features; prefer worktree isolation since intake usually starts from a
  clean default branch.
- **PR-feedback intake** (e.g. a continue-pr skill): review feedback resumes
  the matching change's build phase; if the change is already archived, open
  a new `fix` change whose proposal references the PR.
- PR creation and PR review are NOT part of onto. The workflow ends at a
  verified, closed change on a branch; hand off to the dedicated PR skills
  from there.

## 6. Exit

After routing, the dispatcher is done — the sub-skill owns the phase,
including its gates and exit checklist. Never execute phase work here.

## Gates are sacred

Every sub-skill contains `> **GATE:**` blocks — blocking user decisions.
A gate may only be skipped when the user explicitly pre-answered *that same
question*; a blanket directive (e.g. "run to completion") pre-answers only
the gates that say so, and must be recorded verbatim in
`decisions.directive` in `state.yaml`. When in doubt, stop and ask.
