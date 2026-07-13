## Approach

This is a spec-truth alignment, not a behavior change. Every edit makes a
canonical spec match already-shipped source. No solution comparison is needed;
the only real decision is delete-vs-rewrite for `agent-lifecycle`.

### Decision: delete the `agent-lifecycle` capability, fold its one surviving truth into `config-model`

Evidence (verified 2026-07-13):

- `homonto agents` group not registered — `internal/cli/root.go:28` registers
  `plan/apply/status/doctor/init/import` only.
- `internal/agentlock` and `internal/agentblob` packages do not exist; no
  non-test source references them.
- `[agents.<name>]` parses (`config.go:99` `Agents map[string]Agent`) but the
  loader nils it (`config.go:526` `c.Agents = nil`) — a parse-then-fold
  deprecation shim.

The `agent-lifecycle` spec's Purpose and all of its requirements describe the
imperative command group, the `agentblob` base-blob store, the `agentlock`
prune, and merge-on-`update` — **all removed**. The only surviving true statement
is "`[agents.<name>]` still parses but is folded into `[subagents]` and has no
separate lifecycle." That is a `config-model` fact, not a lifecycle. So:

- **`agent-lifecycle`** → capability retired. The delta removes every requirement;
  archive sync deletes the spec.
- **`config-model`** → the "Lifecycle-managed agents SHALL be declarable as
  `[agents.<name>]`" requirement is rewritten to the deprecation-shim truth.
- **`cli-commands`** → the command-surface requirement drops the `agents` group;
  the agent-subcommand scenarios are removed.

Apply-phase guard: before removing `agent-lifecycle`, grep the specs and source
for any live reference to its requirements (e.g. `internal/merge`, adopt/copy
paths) and confirm they are covered by `apply-pipeline` / `tool-adapters` /
`subagent-projection`, not by `agent-lifecycle`. If a genuine surviving behavior
turns up that no other spec covers, stop and reduce `agent-lifecycle` to that
residue instead of deleting — do not silently drop real behavior.

### CI correspondence check

A coarse gate is sufficient (the review asked only for "even a grep"): extract
every `` `homonto <cmd>` `` token that appears in `openspec/specs/**` as a
command, diff against the command names the CLI registers (parse the literal set
in `internal/cli/root.go`, or run `homonto --help`), and fail if a spec names a
command the binary does not register. Ship it as a small script invoked from the
existing gate (`scripts/gate.sh`) so a tag cannot publish on a weaker check than
a PR. Keep it deliberately dumb — it guards against the exact F5 failure
(spec names a removed command), not full semantic correspondence.

### Out of scope (recorded, not fixed here)

- `config.go:526` silently discarding `[agents]` is an F35-adjacent silent
  no-op; a config-behavior fix is a separate change.
- `docs/superpowers/*` retaining historical designs that mention `homonto
  agents` is F19 (active-only contradiction), not spec truth.
- README `:118` and `using-homonto.md:14` already state the folded truth —
  verify only.
