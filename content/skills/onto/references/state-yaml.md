# state.yaml — canonical schema and template

The agent-managed phase state for one change. This file is the canonical
source; `docs/changes/README.md` summarizes it. **state.yaml is a cache of
truth, not truth** — verifiable file state wins (the same stated-state vs
reality reconciliation homonto's own drift detection performs on tool
configs).

## Template

```yaml
change: <name>             # must equal directory name
workflow: full             # full | fix | tweak
phase: open                # open | design | build | verify | close
created: YYYY-MM-DD
base_ref: <git sha at open — parent of the change's first commit>
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
guides: pending            # pending | updated | "waived: <reason>" (quoted —
                           # a bare waived: <reason> is invalid YAML)
metrics:                   # observational only — never a gate, never blocking
  phases: {}               # <phase>: YYYY-MM-DD stamped at each phase exit
  tasks_total: 0           # finalized at close (checked tasks)
  verify_rounds: 0         # incremented per verify round
  upgraded: false          # a preset→full upgrade happened
archived: false            # set true at archive; phase stays "close"
                           # ("done" is derived-only, never written)
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
| `phase` | the phase-derivation table — never crossing a gate: write the derived phase only if notes.md's Confirmed section records the preceding gate as answered, else the earlier phase (resume at its gate) |
| `created` | date of the oldest commit touching the workspace |
| `base_ref` | parent of the oldest commit touching the workspace (best-effort approximation of the sha at open) |
| `deps` | proposal's `Depends-on:` line, else `[]` |
| `decisions` | null — gates are re-asked; a lost directive is never re-assumed |
| `verify.mode` | null (re-derived at verify entry) |
| `verify.result` | `Result:` line in verification.md, else `pending` |
| `guides` | `pending` unless workspace commits show guide updates |
| `metrics` | phase-advance commit dates, else omitted — best-effort |
| `archived` | false (an archived workspace lives under `archive/`) |
