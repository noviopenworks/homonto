# Comet Design Handoff

- Change: onto-graph-command
- Phase: design
- Mode: compact
- Context hash: 206e419d597eb37353d98e0a29990a6d8ea53804adb4d3052614ea95ab511fae

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-graph-command/proposal.md

- Source: openspec/changes/onto-graph-command/proposal.md
- Lines: 1-35
- SHA256: 5935c003864d1e1b4b86e1009b70f6d6578171ad8fa4840dcc09bc9b43d2784b

```md
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

```

## openspec/changes/onto-graph-command/design.md

- Source: openspec/changes/onto-graph-command/design.md
- Lines: 1-28
- SHA256: c08c46f70805aa7b7e3799db4e2c7b71a05e5055ec91e1e07a261f5ad2fbef15

```md
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

```

## openspec/changes/onto-graph-command/tasks.md

- Source: openspec/changes/onto-graph-command/tasks.md
- Lines: 1-11
- SHA256: 56b33337c22cc1ce2856fd655920534be4203d35f253f467bea1351001048768

```md
# Tasks — onto-graph-command

## 1. onto graph command
- [ ] Add `onto graph [--json]`: enumerate active + archived changes → nodes
      (id/change/phase/archived) + depends-on edges from deps; read-only,
      config-independent; deterministic ordering. TDD: nodes for active+archived,
      a depends-on edge, JSON shape.

## 2. Verify
- [ ] `go test ./internal/ontocli/... -race`, vet, build (incl cmd/onto),
      `openspec validate --all` green.

```

## openspec/changes/onto-graph-command/specs/onto-binary/spec.md

- Source: openspec/changes/onto-graph-command/specs/onto-binary/spec.md
- Lines: 1-26
- SHA256: 1ae4fde88cfb099c687f21d64c25c156423a0880f0ed20d57f4ab8825c9f79e4

```md
# onto-binary

## ADDED Requirements

### Requirement: onto graph emits the change dependency traceability graph

`onto graph` SHALL emit the dependency graph over all onto changes, read-only and
config-independent. It MUST enumerate both active changes (`docs/changes/*`) and
archived changes (`docs/changes/archive/*`), emit one node per change carrying its
stable id, name, phase, and archived flag (a malformed or missing-state change
still appears as a node labeled by its directory, never silently dropped), and
emit one `depends-on` edge for each entry in a change's `deps`. With `--json` it
MUST emit a `{nodes, edges}` object with deterministic ordering; without it, a
readable adjacency listing.

#### Scenario: graph lists active and archived changes with their dependencies

- **GIVEN** an active change depending on an archived change
- **WHEN** `onto graph` runs
- **THEN** both appear as nodes (with id/phase/archived) and a `depends-on` edge
  links the dependent to its dependency

#### Scenario: graph is read-only and needs no config

- **WHEN** `onto graph` runs in a workspace with no `homonto.toml`
- **THEN** it emits the graph without error and mutates no state

```
