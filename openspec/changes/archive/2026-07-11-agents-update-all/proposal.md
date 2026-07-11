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
