---
name: onto
description: onto workflow dispatcher. Use when starting, resuming, or asking about any development work in a repo with the docs/ onto layout — runs tooling preflight, finds the active change, derives the real phase from file state, and routes to the matching onto sub-skill.
---

# onto — Workflow Dispatcher

onto is a five-phase development workflow — **open → design → build → verify →
close** — plus two preset paths (`onto-fix` for bugs, `onto-tweak` for small
non-bug changes). All artifacts live in one `docs/` tree. **Every state
mutation goes through the `onto` binary** (`onto new`, `onto set …`, `onto
advance`, `onto close`): it is the single authority for `onto-state.yaml` and a
hard dependency of these skills — the tooling preflight below fails loudly if it
is missing. The skills never hand-edit the state file. Phase is always
cross-checked against real file state: the state file is a cache of truth, not
truth.

The dispatcher does exactly four things, in order: preflight → discover →
derive → route. It never performs phase work itself.

## 1. Tooling preflight (runs first, every dispatch — warns, never halts)

Run these checks before anything else. A missing tool produces a WARNING
and the workflow proceeds — degraded is still working; the warning tells
the user what they are missing and how to fix it.

0. **onto binary** (required — this is the one hard dependency). Run `onto
   version`. On failure, STOP: the skills drive all workflow state through the
   `onto` binary; without it no phase can mutate state safely. Tell the user to
   install/build it (`go build ./cmd/onto`) before proceeding. This is the only
   preflight check that halts; `rtk` and `graphify` below still warn-never-halt.

1. **rtk** — run `rtk --version`. On success, all subsequent shell
   operations in every onto phase go through rtk (or the rtk hook rewrites
   them transparently). On failure, WARN and proceed: rtk (token-optimized
   CLI proxy) was not found on PATH — token costs will be higher; install
   rtk to reduce them.

2. **graphify** — confirm codebase-understanding tooling is available: the
   `graphify` skill is loadable, or a `graphify-out/` directory or
   `.codegraph/` index exists at the repo root. The open and design phases
   ground every codebase claim in graphify/codegraph queries when they are
   available, rather than guesswork. Indexing is the user's decision: if
   only the skill is available and no index exists, ask the user whether to
   build one before open/design proceeds; if they decline, grounding falls
   back to direct file reading and that fallback is recorded in the
   proposal/design.
   **Staleness counts as absence**: an index older than the recent work
   (rule of thumb: predates the last ~20 commits or is weeks old) gets the
   same ask-to-reindex treatment — ask or proceed, never a halt — and the
   Grounding section records the index's age either way — a confidently
   stale graph is worse than none. If neither the skill nor an index
   exists, WARN, record `grounding: direct file reading (graphify
   unavailable)` in the change's notes.md Grounding section, and proceed.

## 2. Active-change discovery

Scan `docs/changes/*/` excluding `archive/`. A change is active iff its
directory sits directly under `docs/changes/` **and holds a `proposal.md`
or a `state.yaml`**, with `state.yaml` (when present) reading
`archived: false`. A directory with neither artifact is not a change —
skip it (a `templates/`, a scratch dir, an editor folder is not a phantom
active change; never rebuild a state.yaml into it). Also sweep
`docs/changes/archive/*/state.yaml` for `archived: false`: that is a
close interrupted between the `git mv` and the flag — surface it and
finish the interrupted archive (the moved workspace still needs its
`archived: true` flag the halted `onto close` never wrote).
If a change carries an `abandoned:` reason it is retired — never list it
as active.

| Active changes | User input | Behavior |
|---|---|---|
| None | description given | Route to `onto-open` with the description |
| None | nothing | Ask what the user wants to work on, then `onto-open` |
| Exactly one | nothing | Resume it: derive phase, route |
| Exactly one | new description | ASK: continue the active change or open a new one |
| Two or more | anything | LIST them (name, workflow, claimed phase, deps status) and ASK which to resume before doing anything else |

**Dependencies**: each change's `state.yaml` may name `deps:` — changes
that must archive before this one builds. A dep counts as **archived iff
a directory `docs/changes/archive/????-??-??-<dep>/` exists** — the
date-anchored exact-name match (`YYYY-MM-DD-` prefix per the archive
contract), never a bare suffix match, which falsely resolves deps whose
name is the tail of another change's name. **An active workspace with the
dep's name overrides any archive hit** (a reused name in flight is not
archived). Discovery listings show deps status (`ready` /
`blocked by <name>`). Before resuming a change whose deps are not all
archived, warn and require an explicit user choice: proceed anyway,
switch to the dependency, or stop. Two findings to surface immediately:
a dep matching **no active and no archived change** (ask the user to
correct or drop it), and a dep chain that **reaches the current change —
including a self-dep or an A⇄B cycle** (unsatisfiable by construction;
ask the user to break the cycle). For multiple simultaneously active
changes, recommend one git worktree per change — coupled work that can't
be separated should have been one change (the split-preflight rule
already says so). **Close them one at a time**, though: two closes running
at once both merge into shared `docs/specs/*` and both draw ADR numbers
from the same `docs/adr/` (onto-close re-scans before each move to avoid a
clobber, but serial closes remove the race outright).

If the repo has no `docs/changes/` tree at all, offer to bootstrap the
layout: create `docs/{adr,specs,changes/archive,guides}/`, writing
`docs/changes/README.md` from `references/changes-readme.md` and
`docs/specs/README.md` from `onto-close/references/specs-readme.md` (the
`docs/adr/` numbering contract and `docs/guides/` are conventional). Then
proceed to `onto-open`.

## 3. Phase derivation and cross-check

`state.yaml` is a **cache of truth, not truth**. On every dispatch:

1. Read `state.yaml`. Its canonical schema, template, and per-field rebuild
   rules live in `references/state-yaml.md` in this skill's directory —
   **the single source**. `docs/changes/README.md`, when the repo has one,
   points here rather than copying, so the two never drift. If a skill's
   `references/` directory is genuinely missing, say so, fall back to
   reading this SKILL.md's own tables, and continue — degrade, never halt;
   but note that a reconstructed lint or grammar is weaker than the real
   one, so flag any close run made without them.
2. Independently derive the phase from artifacts with this table
   (**first match from the top wins — strongest evidence first**; this
   table is authoritative — any repo README points here, never re-states
   it):

| Evidence | Real phase |
|---|---|
| `archived: true` or workspace under `archive/` | done |
| `design.md` marked `Status: Under revision` | design |
| `verification.md` with a `Result: pass` line | close |
| `tasks.md` contains ≥1 task and all are checked | verify |
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
   normally. **One exception: the verify→close boundary has no gate** (the
   failure gate fires only on a fail). So a `phase: verify` claim beside a
   `verification.md` reading `Result: pass` is not an unanswered gate — it
   is a lagging write. Advance `phase` to `close` and route to `onto-close`
   without re-verifying; the pass already stands in the file. Re-running
   verify here would only discard fresh evidence the report already holds.
4. **Cross-check `workflow` too, not just phase.** Resolve it in this
   priority order and stop at the first that applies:
   1. The proposal's `Preset:` marker. An upgrade annotation
      (`Preset: fix (upgraded to full YYYY-MM-DD)`) means **full**.
   2. A `Status: Confirmed` (or `Under revision`) `design.md` means
      **full** — a designed change has a lifecycle no branch name can
      strip, so the branch prefix is ignored here.
   3. The branch prefix (`fix/`, `tweak/`) — only when neither 1 nor 2
      applies (the branch belongs to the checkout, not the change; a
      leftover `fix/` branch must not demote a real change).
   4. Otherwise **full** (a detached HEAD or non-prefixed branch is no
      signal).

   On mismatch the file sources win — correct, announce, reroute — with
   one hard asymmetry: **a correction may upgrade (preset→full) silently,
   but a downgrade (full→preset) never happens without fresh user
   confirmation.** Never talk a change down.
5. A missing or malformed `state.yaml` is never an error: rebuild it per
   the per-field table in `references/state-yaml.md` (`workflow` from the
   proposal's `Preset:` marker incl. upgrade annotation, else the branch
   prefix, else `full`; `base_ref` = parent of the oldest commit touching
   the workspace; `decisions` reset to null so gates are re-asked;
   `verify.result` from verification.md's `Result:` line; `deps` from the
   proposal's `Depends-on:` line; `metrics` best-effort, never blocking),
   announce the rebuild, continue. **Rebuild never crosses a gate**: cap
   the derived phase per the boundary table in
   `references/state-yaml.md` — open→design and design→build need their
   notes.md Confirmed records, build→verify needs the plan-ready record,
   verify→close is decidable from verification.md's `Result: pass` alone;
   demote one boundary at a time, floor `open` (full) / `build` (presets).
   A lost state file must not skip what the user never confirmed.
6. Never trust conversation history for phase detection — after context
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
copy/config/docs/prompt touch-up or a small feature within tweak limits
(≤5 files excluding tests, no new capability, no existing-spec requirement
change) → `onto-tweak`; anything needing design → `onto-open` (full).
Preset skills contain upgrade rules that force the full path when scope
grows — never talk a change *down* from full to a preset.

**Reopen and abandon** (both need explicit user intent):

- **Reopen** — a defect found after verify passed but *before* archive:
  route to build. Add tasks for the fix in `tasks.md` and run `onto set
  verify-result <name> pending`; flip `verification.md`'s `Result:` line to
  `Result: superseded (reopened <date>)`. The unchecked tasks plus the
  invalidated result drive the dispatcher's derivation back to build — no
  phase field is written. A defect in an *archived* change is new work — open
  a fresh `fix` change whose proposal references the archived one; archives
  are never edited.
- **Abandon** — the user drops a change: there is no `onto abandon` command
  (deferred to N2), so this is the single sanctioned direct state note. Add
  `abandoned: "<reason>"` (the user's words) to `onto-state.yaml`, then `onto
  close <name>` to move the workspace to `docs/changes/archive/YYYY-MM-DD-<name>/`
  and set `archived: true`, in one commit. It leaves the active list and never
  routes anywhere again. No spec merge, no ADR numbering.

  (`onto close` requires `phase: close`; an abandoned change may be at any
  phase. If `onto close` refuses, fall back to the manual `git mv` +
  `archived: true` note — record which was used. This residual is the flagged
  N2 gap.)

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

## 7. Delegation, parallelization, and dialogs

The onto framework ships two read-only **specialist subagents** — they install
with onto and the phases delegate to them. They investigate and report back, and
run as independent agents, so several can run **in parallel**. Both tools support
this: **OpenCode** dispatches subagents as child sessions and **Claude Code**
runs them as parallel Task-tool agents (send multiple Task calls in one turn).

| Subagent | Use it to | Delegated from |
|---|---|---|
| `codebase-explorer` | answer "how does X work / where does behavior live" by reading across many files, returning conclusions not dumps | open, design, and any phase needing grounding |
| `code-reviewer` | review a diff for correctness, security, contract, and clarity, ranked by severity | build (per task) and verify |

**Delegate, and fan out.** When a phase needs investigation or review, hand it to
the subagent rather than doing it inline — and when the questions are
**independent**, dispatch them **concurrently** (one subagent invocation per
question) instead of serially:

- **design / grounding** — split a broad "how does this subsystem work" into
  several targeted `codebase-explorer` tasks and run them at once; synthesize the
  returns into the design.
- **build** — after each task's edits, hand the diff to `code-reviewer`.
  Independent tasks that touch **non-overlapping** files can be explored/reviewed
  in parallel; tasks that share files stay serial (one commit each, in order).
- **verify** — delegate the change-wide diff audit to `code-reviewer` while you
  check spec scenarios.

The orchestrator (this session) still owns every edit, commit, and the `onto`
binary calls — the subagents only read and report. Never let a subagent mutate
workflow state.

**Dialogs — prefer them, in either tool.** Whenever a `> **GATE:**` block or any
either/or decision comes up, ask it through an **interactive choice dialog** — a
clear prompt, a short header, and the concrete choices — rather than burying the
question in prose. It is faster for the user and records a definite answer. Both
tools have a dialog mechanism, so use it in both:

- **OpenCode** — the **question** tool (the shipped subagents allow it via
  `permission.question`).
- **Claude Code** — the **AskUserQuestion** tool.

Fall back to a plain written question only when neither is available. A dialog
never *replaces* a gate — it is how the gate is asked.

## Gates are sacred

Every sub-skill contains `> **GATE:**` blocks — blocking user decisions.
A gate may only be skipped when the user explicitly pre-answered *that same
question*; a blanket directive (e.g. "run to completion") pre-answers only
the gates that say so, and must be recorded verbatim via `onto set directive
<name> "<text>"`. When in doubt, stop and ask.

## Prose discipline (every artifact)

onto writes prose a human reads later: `proposal.md`, `design.md`, `notes.md`,
ADR drafts, `verification.md`, guide updates, and commit messages. Run the
**onto-no-slop** skill (bundled with this framework) over each prose artifact
before its phase gate — cut filler and adverbs, use active voice, name the
actor, be specific, vary the rhythm, no em dashes. Record the score in
`notes.md` (`no-slop: <artifact> <total>/50`; below 35 means revise before the
gate) — the checkbox is worth nothing without the number behind it.

It edits prose, never contract. Machine-read markers (`Status:`, `Result:`,
`Preset:`, checkbox syntax, `SHALL`/`MUST` lines, GIVEN/WHEN/THEN), a
requirement's normative wording, and mandated template structure are off-limits
— rewording one breaks derivation or the lint. Keep load-bearing terms and
genuine distinctions; drop the empty adverb and the manufactured reversal. Each
phase's exit checklist re-states this.
