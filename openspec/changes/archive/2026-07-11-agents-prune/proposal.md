## Why

`agents doctor` reports two kinds of stale installs — an **orphan** (an agent in
the lockfile no longer declared in the config) and a **de-declared target** (a
target recorded for an agent that the agent no longer targets) — but nothing
removes them. This change adds `homonto agents prune` to clean them up safely,
completing the lifecycle: `add` installs, `doctor` detects drift, `prune` removes
what you removed from the config.

## What Changes

- Add `homonto agents prune`: removes homonto-managed agent installs that are no
  longer declared, and drops their lockfile records.
  - **Orphan agent** (in `.homonto/agents-lock.json`, not in the config): each of
    its recorded target install files is removed and the agent's lockfile entry
    is dropped.
  - **De-declared target** (a target in an agent's `Installed` that the agent no
    longer targets): that target's install file is removed and its `Installed`
    entry dropped; the agent (and its still-declared targets) is kept.
  - **Safety**: only a file at a homonto-*recorded* install path is touched. A
    file whose on-disk content differs from the recorded base hash (a local edit)
    is backed up to `<path>.bak` before removal — no user edit is silently lost.
    A pruned target's leftover `<path>.merged` conflict sidecar is also removed.
  - It reports each pruned item (and any backup), saves the lockfile, and prints
    `nothing to prune` when the workspace is already clean.
- `--dry-run` lists what would be pruned without changing anything.
- Register `prune` under `agentsCmd()`. `homonto agents` gains add / doctor /
  list / prune / update.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents prune`, which removes homonto-managed
  installs for orphaned agents and de-declared targets (backing up any
  locally-modified file first) and drops their lockfile records; `--dry-run`
  previews.

## Impact

- `internal/cli/agents.go`: new `prune` subcommand (`agentsPruneCmd`), reusing
  `agentlock` and the existing sorted/hash helpers.
- Tests in `internal/cli`.
- No new dependency. `add`/`list`/`doctor`/`update` unchanged.
- Deferred: blob GC (unreferenced `.homonto/agents-blobs/*`) — a separate
  concern; prune does not GC blobs (they may be shared / content-addressed).
