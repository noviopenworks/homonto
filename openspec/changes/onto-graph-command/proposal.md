# onto graph: emit the change dependency traceability graph

## Why

Roadmap X1 (the typed traceability graph), building on the stable-id core
(`onto-stable-change-id`). onto changes declare `deps` (other change names), but
nothing surfaces the relationships — the toolkit cannot answer "what depends on
this change" or show the dependency DAG. With stable ids now present, a first
traceability graph is a small, concrete step: enumerate every change (active and
archived), emit its node (stable id, name, phase, archived) and its
`depends-on` edges.

## What Changes

- Add `onto graph [--json]`: reads `docs/changes/*` (active) and
  `docs/changes/archive/*` (archived), and emits a graph of **nodes** (one per
  change: `id`, `change`, `phase`, `archived`) and **edges** (`depends-on`, one
  per entry in each change's `deps`). Text output is a readable adjacency listing;
  `--json` emits `{nodes:[…], edges:[{from,to,type}]}` for tooling.
- Read-only, config-independent (like `onto status`): it never mutates state or
  needs a `homonto.toml`.

## Impact

- **Specs:** `onto-binary` gains a requirement that `onto graph` emits the
  change dependency graph over active and archived changes.
- **Behavior:** additive new command; nothing else changes.
- **Risk:** low — a read-only enumerator; Go tests pin nodes/edges/JSON shape.

## Non-goals

- The full typed-edge set (`implements`/`tests`/`supersedes`/`deviates-from`/
  `released-in`) that would link changes to code, specs, and releases — a larger
  follow-on; this delivers the `depends-on` edge over changes.
- CI validation of the graph; the comet/OpenSpec flow.
