# state.yaml — canonical schema and template

The agent-managed phase state for one change. This file is the canonical
source (`docs/changes/README.md`, when the repo has one, points here rather
than restating it). **state.yaml is a cache of
truth, not truth** — verifiable file state wins (the same stated-state vs
reality reconciliation homonto's own drift detection performs on tool
configs).

## Template

```yaml
change: <name>             # must equal directory name
workflow: full             # full | fix | tweak
phase: open                # open | design | build | verify | close
created: YYYY-MM-DD
base_ref: <git rev-parse HEAD, captured when open creates the workspace,
           before the workspace commit — written once, never recomputed>
deps: []                   # change names that must archive before this builds
decisions:                 # null until chosen (build entry; presets default at open-lite)
  isolation: null          # branch | worktree
  execution: null          # direct | subagent
  tdd: null                # tdd | direct
  directive: null          # verbatim user pre-authorization text, if any
verify:
  mode: null               # light | full (set at verify entry by scale rules)
  result: pending          # pending | pass | fail (accepted deviations live in
                           # verification.md; the enum stays "pass")
close:
  merged: false            # set true by onto-close before it merges deltas /
                           # numbers ADRs, so an interrupted close re-enters
                           # idempotently (skips the merge on the second pass)
guides: pending            # pending | updated | "waived: <reason>" (quoted —
                           # a bare waived: <reason> is invalid YAML)
metrics:                   # observational only — never a gate, never blocking
  phases: {}               # <phase>: YYYY-MM-DD stamped at each phase exit
  tasks_total: 0           # finalized at close (checked tasks)
  verify_rounds: 0         # incremented per verify round
  upgraded: false          # a preset→full upgrade happened
archived: false            # set true at archive; phase stays "close"
                           # ("done" is derived-only, never written)
# abandoned: "<reason>"    # only on user-abandoned changes, written together
                           # with archived: true at the abandon step; the
                           # reason is the user's words, never invented
```

## Field rules

- `phase` advances only when a phase's exit gate is answered — never
  because artifacts happen to exist (gates win upward).
- `decisions.directive` holds blanket pre-authorizations verbatim; it
  pre-answers only the gates whose skill text says MAY be pre-authorized.
- `deps` names other changes under `docs/changes/`; a dep is archived iff
  `docs/changes/archive/????-??-??-<dep>/` exists (date-anchored exact
  name, never bare suffix), and an active workspace with the dep's name
  overrides any archive hit. The dispatcher warns before resuming a change
  whose deps are not all archived; a dep matching no active or archived
  change, a self-dep, or a dep cycle reaching this change are findings to
  correct or drop.
- `metrics` is best-effort observational data. Skills stamp
  `metrics.phases.<phase>` on exit; close finalizes the rest. Never block
  on metrics for any reason.

## Rebuild rules (never an error)

Rebuild applies at two granularities: a **missing or unparseable file** is
rebuilt whole from the table below; an **individually missing or
ill-typed field** in an otherwise valid file is rebuilt per its table row
alone, other fields untouched (e.g. a pre-v2 state.yaml without `deps` is
treated as `deps: []`; a string-typed `deps` is re-read from the
proposal's `Depends-on:` line). Field-level repair never resets
`decisions` — gates are only re-asked when the whole file was lost.

| Field | Rebuild from |
|---|---|
| `change` | directory name |
| `workflow` | proposal's `Preset:` marker — an upgrade annotation (`Preset: fix (upgraded to full YYYY-MM-DD)`) means `full` — else branch prefix (`fix/`,`tweak/`), else `full` |
| `phase` | the phase-derivation table, gate-capped per the boundary table below — a rebuild never crosses an unanswered gate |
| `created` | date of the oldest commit touching the workspace (every phase commits the workspace at exit, so this is the open-exit commit) |
| `base_ref` | parent of the oldest commit touching the workspace — exact, not approximate, because open commits the workspace at exit, making that commit's parent the HEAD open captured |
| `deps` | proposal's `Depends-on:` line, else `[]` |
| `decisions` | null — gates are re-asked (full workflow); presets re-default (`isolation: branch`, `execution: direct`, fix `tdd: tdd`, tweak `tdd: direct`) since their values were never gate answers; a lost directive is never re-assumed |
| `verify.mode` | null (re-derived at verify entry) |
| `verify.result` | `Result:` line in verification.md — a `superseded` line rebuilds as `pending` (the revision invalidated it) — else `pending` |
| `close.merged` | `false` — a rebuilt state cannot tell whether a prior close merged, so re-derive conservatively: if living specs already contain this change's requirements, the merge landed (skip it); the post-merge lint's duplicate check is the backstop |
| `guides` | `pending` unless workspace commits show guide updates |
| `metrics` | phase-advance commit dates, else omitted — best-effort |
| `archived` | false (an archived workspace lives under `archive/`) |

## Gate caps for phase rebuild (boundary → record consulted)

| Boundary | Gate | Decidable from |
|---|---|---|
| open → design | artifact review | notes.md Confirmed entry (onto-open exit mandates recording it) |
| design → build | approach confirmation | notes.md Confirmed entry (onto-design records it) |
| build → verify | plan-ready | notes.md Confirmed entry (onto-build's gate records its answer there — decisions in the lost file don't survive) |
| verify → close | none (failure gate fires only on fail) | `verification.md` with `Result: pass` — no demotion when present |

Rules: demote **one boundary at a time** until the boundary's record is
found (or the floor). The floor is `open` for full workflow and **`build`
for presets** (presets have no open/design phases; their open-lite
confirmations are re-defaulted, not re-asked, at build entry). With
notes.md missing entirely, demotion iterates through every notes-dependent
boundary down to the floor — only verify→close resists, staying decidable
from verification.md regardless.

**Upgraded presets** (a `Preset: … (upgraded to full …)` marker) never
floor at `open`: the upgrade annotation is itself the open→design record
(the user confirmed the upgrade), and a `design.md` marked
`Status: Confirmed` is itself the design→build record. Their floor is
`design` without a confirmed design.md, `build` with one. A rebuild must
not send a change with a user-confirmed design back to clarification.
