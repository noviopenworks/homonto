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
