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
