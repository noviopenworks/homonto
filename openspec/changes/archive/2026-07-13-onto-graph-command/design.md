# Design — onto graph

## Command

`onto graph [--json]` (read-only, config-independent, mirroring `onto status`).
Enumerate `docs/changes/*` (skip the `archive` dir) as active and
`docs/changes/archive/*` as archived; for each, `ontostate.Classify(dir)` (or
Load) → a node `{ID, Change, Phase, Archived}`; a malformed/missing-state change
still yields a node labeled by directory (never silently dropped, mirroring the
status F14 rule). For each change's `st.Deps`, emit an edge `{From: change,
To: dep, Type: "depends-on"}`.

## Output

- text: `<change> (<id>, <phase><, archived>)` then `  → depends-on <dep>` lines;
  a change with no deps prints just its node line.
- `--json`: `{"nodes":[{"id","change","phase","archived"}],"edges":[{"from","to","type"}]}`
  with stable (sorted) ordering for deterministic output.

## Risk

Low — read-only enumeration reusing the status classification. Go tests build a
few changes with deps and assert the node set, the depends-on edges, and the JSON
shape.

## Alternatives
- Resolve deps to ids in the edges — deferred; deps are recorded as names today,
  so edges carry names (a follow-on can map to ids once deps are id-keyed).
