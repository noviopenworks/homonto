# docs/changes/README.md — canonical template

Bootstrap writes this into a repo at `docs/changes/README.md` when it
creates the onto layout. It is a **pointer**, not a second copy of the
rules — the onto skills stay the single source, so the two never drift.

## Template

```markdown
# Changes

Active and archived onto change workspaces.

- **Active change**: a directory directly under `docs/changes/` holding a
  `proposal.md` or `state.yaml`, with `state.yaml` reading
  `archived: false`. A directory with neither artifact is not a change.
- **Archived change**: under `docs/changes/archive/YYYY-MM-DD-<name>/`,
  `state.yaml` `archived: true`. Archives are history — never edited, with
  the single exception of `ship.md`.

## State model

`state.yaml` is the per-change phase cache. Its canonical schema, field
rules, rebuild rules, and the phase-derivation table are defined by the
onto skill set (the `onto` dispatcher skill and its
`references/state-yaml.md`) — this README does not restate them, so there
is nothing to keep in sync. The dispatcher re-derives the real phase from
file state on every run; `state.yaml` is a cache, and files win.

## Layout contract

- `docs/changes/<name>/` — one active change: `state.yaml`, `notes.md`,
  `proposal.md`, `tasks.md`, and (full workflow) `design.md`, `plan.md`,
  `adr/`, `specs/`, `verification.md` as the phases produce them.
- `docs/changes/archive/YYYY-MM-DD-<name>/` — a closed or abandoned change.
```
