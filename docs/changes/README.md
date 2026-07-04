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
| `notes.md` | open (updated through design) | context-loss checkpoint: confirmed facts, pending items |
| `plan.md` | build | implementation plan (full workflow) |
| `verification.md` | verify | evidence-based verification report |
| `validation-notes.md` | build/verify (optional) | captured validation evidence (dry-runs, command output) |
| `ship.md` | close (optional, post-archive) | accepted ship-handoff PR body |

A change is **active** iff its directory sits directly under `docs/changes/`
(not under `archive/`) and `state.yaml` has `archived: false` (or is absent —
the dispatcher rebuilds it).

## Archive contract

`onto-close` moves the whole workspace to
`docs/changes/archive/YYYY-MM-DD-<name>/` (date = close date), unmodified
except `archived: true` in `state.yaml`. Archived changes are history — never
edited afterwards, with exactly one sanctioned exception: an accepted
ship handoff adds `ship.md` immediately after archiving.

## state.yaml schema

Canonical schema, template, and per-field rebuild rules live with the
dispatcher skill: `onto/references/state-yaml.md`. Summary:

```yaml
change: add-foo            # must equal directory name
workflow: full             # full | fix | tweak
phase: build               # open | design | build | verify | close
created: 2026-07-04
base_ref: <git sha at open>
deps: []                   # change names that must archive before this builds
decisions:                 # null until chosen (build entry; presets default at open-lite)
  isolation: branch        # branch | worktree
  execution: direct        # direct | subagent
  tdd: tdd                 # tdd | direct
  directive: null          # verbatim user pre-authorization text, if any
verify:
  mode: null               # light | full (set at verify entry by scale rules)
  result: pending          # pending | pass | fail (deviations accepted at the
                           # verify gate are recorded in verification.md;
                           # result stays "pass")
guides: pending            # pending | updated | "waived: <reason>" (quoted —
                           # a bare waived: <reason> is invalid YAML)
metrics:                   # observational only — stamped at phase exits,
  phases: {}               # finalized at close; never a gate, never blocking
  tasks_total: 0
  verify_rounds: 0
  upgraded: false
archived: false            # set true at archive; phase stays "close" ("done"
                           # is derived-only, never written)
```

## Lifecycle and recovery rules

- The agent edits this file directly — there are no scripts.
- **state.yaml is a cache of truth, not truth.** Verifiable file state wins.
- On every `/onto` dispatch the phase is re-derived from artifacts using the
  table below and cross-checked. If the derived phase is **earlier** than
  the claimed phase, files win: correct state.yaml, announce the correction,
  resume at the derived phase. If the derived phase is **later** than the
  claimed phase, do not silently promote — the phase field is advanced only
  when a phase's exit gate is answered, so a lagging claim means an
  unanswered gate: resume at the claimed phase's gate (artifacts already
  prepared) and let it advance normally. Gates are never skipped because
  artifacts happen to exist.
- An explicit user directive that pre-answers a gate (e.g. "run to
  completion") is recorded **verbatim** in `decisions.directive`.
- A missing or malformed state.yaml is rebuilt instead of failing, per the
  per-field rebuild table in `onto/references/state-yaml.md` (key rules:
  `workflow` from the proposal's `Preset:` marker → branch prefix →
  `full`; `base_ref` = parent of the oldest workspace commit; `decisions`
  = null so gates are re-asked; `verify.result` from verification.md's
  `Result:` line; `deps` from the proposal's `Depends-on:` line;
  `metrics` best-effort from phase-advance commit dates, never blocking).

## Phase derivation (first match from the top wins — strongest evidence first)

| Evidence | Real phase |
|---|---|
| `archived: true` or workspace under `archive/` | done |
| `design.md` marked `Status: Under revision` | design |
| `verification.md` with a `Result: pass` line | close |
| `tasks.md` contains ≥1 task and all are checked | verify |
| `design.md` marked `Status: Confirmed`, or a preset workspace | build |
| `proposal.md` + `tasks.md` exist (full workflow, no confirmed design) | design |
| workspace exists, artifacts incomplete | open |
