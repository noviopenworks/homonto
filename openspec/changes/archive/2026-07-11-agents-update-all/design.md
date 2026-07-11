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
