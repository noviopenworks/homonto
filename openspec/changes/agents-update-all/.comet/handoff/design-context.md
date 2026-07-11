# Comet Design Handoff

- Change: agents-update-all
- Phase: design
- Mode: compact
- Context hash: 8d6d410776c93b8c96cf4e0c83acbb3e56bdaf0dcbf32b0e9a0d22b262415b41

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-update-all/proposal.md

- Source: openspec/changes/agents-update-all/proposal.md
- Lines: 1-51
- SHA256: f8d722e3f5b675d01c052eccb46d15804dfd59138f828f0f3e41261f683591e6

```md
## Why

`agents update <name>` (v2 #5b) three-way-merges one installed agent. The
approved design's final merge slice (#5c) adds `agents update --all`: run that
same merge across every installed agent and summarize the outcome — the bulk
"reconcile all my agents with their sources" convenience that the roadmap's
`migrate` calls for (a thin wrapper over the per-agent merge, not a new
algorithm).

## What Changes

- Add an `--all` flag to `homonto agents update`. `agents update --all` (with no
  agent name) runs the three-way merge over **every installed agent** recorded in
  `.homonto/agents-lock.json`, and prints a summary: how many were merged/updated,
  up-to-date, conflicted, or skipped.
  - An agent still declared in the config is merged exactly as `agents update
    <name>` would (auto-merge / `.merged` sidecar on conflict / base advance).
  - An installed agent no longer declared in the config (orphan) is skipped with
    a note (it is `doctor`'s concern, not `update`'s).
  - A per-agent failure (e.g. a missing local source file) is reported for that
    agent and does not abort the rest of the run.
  - The command exits non-zero if any agent had a conflict or a per-agent error;
    it exits 0 when all agents are clean.
- `agents update` with neither `--all` nor a name, or with both, is a clear usage
  error. `agents update <name>` (single) behavior is unchanged.
- Internally, the per-agent update body is refactored into a reusable helper so
  the single and `--all` paths share exactly one merge implementation.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: `homonto agents update` gains `--all`, which three-way-merges
  every installed agent against its source and summarizes the result (the
  `migrate`/bulk-reconcile convenience), exiting non-zero if any conflict or
  per-agent error occurs.

## Impact

- `internal/cli/agents.go`: extract the per-agent update logic into a helper;
  `agentsUpdateCmd` gains `--all` (with arg/flag validation) and the aggregate
  loop.
- Tests in `internal/cli`.
- No new dependency. Single `agents update <name>`, and all other commands,
  behave as before.
- Deferred: a `migrate` alias command (documented equivalence to `update --all`),
  `--markers` in-file conflict mode, builtin/remote sources.

```

## openspec/changes/agents-update-all/design.md

- Source: openspec/changes/agents-update-all/design.md
- Lines: 1-71
- SHA256: 32a166db4eb14e66febd887339341587f5689453b5c82738024321a007810810

```md
## Context

v2 #5c — the bulk merge convenience (the roadmap's `migrate`), a thin wrapper over
the #5b per-agent three-way merge. Approved design:
docs/superpowers/specs/2026-07-11-agents-3way-merge-design.md.

## Goals / Non-Goals

**Goals**: `agents update --all` runs the per-agent merge over every installed
agent, summarizes, exits non-zero on any conflict/error; refactor the per-agent
body into one shared helper.

**Non-Goals**: a separate `migrate` command (documented as update --all);
`--markers`; builtin/remote; changing single `update <name>` behavior.

## Decisions

### D1 — Extract `runAgentUpdate` helper

Refactor the current `agentsUpdateCmd` RunE body (everything after loading config,
lock, home) into:
`func runAgentUpdate(cmd *cobra.Command, name string, c *config.Config, lock *agentlock.Lock, cfgDir, homontoDir, home string) (conflicted bool, err error)`.
- It does the existing per-agent work: lookup (undeclared→err), non-local→err,
  source-read→err, per-target merge (D1 of #5b), mutating `lock.Agents[name]`
  (conflicted targets keep prev), printing per-target statuses.
- It returns `conflicted` and does NOT call `lock.Save` (the caller saves once).
Both `update <name>` and `update --all` call it; the SINGLE path keeps today's
semantics (err propagates; conflicted → non-zero summary).

### D2 — `agents update` arg/flag validation

Add `--all` bool flag. `Args: cobra.ArbitraryArgs`. In RunE:
- if `all && len(args)>0` → usage error "cannot combine --all with an agent name".
- if `!all && len(args)!=1` → usage error "provide an agent name or --all".
- if `!all`: `runAgentUpdate` for `args[0]`; on err return; `lock.Save`; if
  conflicted → non-zero summary. (Unchanged behavior.)
- if `all`: iterate `sortedKeysAgents(lock.Agents)`:
  - if name not in `c.Agents` (orphan) → print `"<name>: skipped (no longer declared)"`; continue.
  - else `conf, err := runAgentUpdate(...)`; if err → print `"<name>: error: <err>"`, set `hadError=true`, continue; else `anyConflict = anyConflict || conf`.
  - track counts (processed, conflicted, skipped, errored).
  - After the loop: `lock.Save(homontoDir)`; print a summary line
    `"agents update --all: N processed, C conflicted, S skipped, E errored"`; if
    `anyConflict || hadError` → return a non-zero summary error.

### D3 — Per-agent error isolation in --all

`runAgentUpdate` returns `err` for hard per-agent problems (missing source file,
non-local — though non-local can't occur for lockfile agents, all installed are
local). In `--all`, an err is captured per agent (printed, `hadError`), never
aborts the loop. In single mode, err propagates (unchanged). A conflict is a
normal per-agent outcome (not an err) → captured via `conflicted`.

## Risks / Trade-offs

- **Refactor risk**: extracting the helper must preserve the exact #5b single-
  update behavior. The existing `update` tests (disjoint/conflict/idempotent/
  fallback/foreign-file/etc.) MUST still pass unchanged — the guard against a
  regression.
- **Partial --all**: some agents merged, one conflicted → lock saved with the
  clean ones advanced, conflicted kept on prev, exit non-zero. Re-run after
  resolving is a no-op for the clean ones. Consistent with single update.
- **Orphan skip**: `--all` doesn't prune orphans (that's a future concern);
  doctor reports them.

## Migration Plan

Additive flag. No migration.

## Open Questions

None — approved. A `migrate` alias command is a documented, deferred nicety.

```

## openspec/changes/agents-update-all/tasks.md

- Source: openspec/changes/agents-update-all/tasks.md
- Lines: 1-12
- SHA256: 947d0653faa704f445823e25f69d7999e4cb87efca79a0b300486d11608c4973

```md
## 1. `agents update --all` (`internal/cli`)

- [ ] 1.1 (TDD RED first) Refactor per Design Doc D1: extract `runAgentUpdate(cmd, name, c, lock, cfgDir, homontoDir, home) (conflicted bool, err error)` from the current `agentsUpdateCmd` body (does the per-agent merge, mutates lock.Agents[name], prints per-target statuses, does NOT Save). The existing single-update tests must still pass unchanged.
- [ ] 1.2 (TDD RED first) Add `--all` bool flag + `cobra.ArbitraryArgs` + validation (D2): `all && args>0` → usage err; `!all && args!=1` → usage err; single path calls the helper then Save then conflicted→non-zero (unchanged); `--all` path loops `sortedKeysAgents(lock.Agents)` — orphan (not in config)→skip note; else helper (err→print+hadError, else anyConflict); Save once; print summary; return non-zero if anyConflict||hadError.
- [ ] 1.3 (TDD RED first) Tests: `update --all` with one disjoint-mergeable + one up-to-date agent → both processed (first merged, second up-to-date), summary, exit 0; one conflicting agent → its `.merged` written + exit non-zero, other still processed; orphan (in lock, not config) → skipped note, exit 0 (absent other issues); `update <name> --all` and `update` (no name, no --all) → usage errors; single `update <name>` still works (all prior update tests green).
- [ ] 1.4 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update --all' bulk-merges every installed agent`

## 2. Regression and docs

- [ ] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): two installed agents, edit one's source disjointly → `agents update --all` merges it, reports the other up-to-date, exit 0; make one conflict → `update --all` writes its `.merged`, exit non-zero, other processed.
- [ ] 2.2 Update `docs/roadmap.md` v2 status (update --all landed; migrate = update --all) + README (mention `agents update --all`). No over-claim.
- [ ] 2.3 Commit all changes.

```

## openspec/changes/agents-update-all/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-update-all/specs/agent-lifecycle/spec.md
- Lines: 1-37
- SHA256: 50d692c36304c8b1355cd7de0caa41f27b19241f00f464abab9b65afffaa5b72

```md
## ADDED Requirements

### Requirement: homonto agents update --all reconciles every installed agent

`homonto agents update --all` SHALL run the three-way merge (the same as `homonto
agents update <name>`) across every agent recorded in `.homonto/agents-lock.json`,
and print a summary of the outcome (merged/updated, up-to-date, conflicted,
skipped). An agent still declared in the config SHALL be merged exactly as the
single-agent update would; an installed agent no longer declared in the config
SHALL be skipped with a note; a per-agent error (e.g. a missing local source)
SHALL be reported for that agent without aborting the rest. The command SHALL exit
non-zero if any agent had a conflict or a per-agent error, and exit 0 when all
agents are clean. `agents update` with neither a name nor `--all`, or with both,
SHALL be a usage error; single `agents update <name>` behavior is unchanged.

#### Scenario: update --all merges every installed agent

- **GIVEN** two installed agents, one with a disjoint local+source edit and one already up-to-date
- **WHEN** `homonto agents update --all` runs
- **THEN** the first is auto-merged and the second reported up-to-date, a summary is printed, and the command exits 0

#### Scenario: update --all exits non-zero on any conflict

- **GIVEN** two installed agents, one of which has an overlapping (conflicting) edit
- **WHEN** `homonto agents update --all` runs
- **THEN** the conflicting agent gets a `.merged` sidecar and the command exits non-zero, while the other agent is still processed

#### Scenario: update --all skips an orphaned agent

- **GIVEN** an installed agent that is no longer declared in the config
- **WHEN** `homonto agents update --all` runs
- **THEN** it is skipped with a note and does not cause the whole run to fail (absent other issues, exit 0)

#### Scenario: name and --all are mutually exclusive

- **WHEN** `homonto agents update <name> --all` runs (or `agents update` with neither)
- **THEN** it is a usage error

```
