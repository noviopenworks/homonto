## Why

`agents doctor` (v2 #3) reports when an installed agent has drifted — its
`local:` source file changed, or its installed copy was modified/deleted on disk.
The complementary *action* is missing: re-materializing the agent from its
current source so the install matches the source again. This change adds
`homonto agents update` — the fix for the drift `doctor` detects. Because homonto
is declarative (the config is the source of truth and homonto never edits
`homonto.toml`), version *pinning* is simply editing `[agents.<name>].version` in
the config, so no separate `pin` command is needed; `update` is the real lifecycle
mutation. To protect user work, a locally-modified install is **backed up** before
being overwritten (full three-way-merge is a later increment).

## What Changes

- Add `homonto agents update <name>`: re-installs an already-installed declared
  agent from its current source, refreshing `.homonto/agents-lock.json`.
  - The agent must be declared and already installed (in the lockfile); an
    uninstalled agent returns an error pointing at `agents add`. `local:` sources
    only (builtin/remote deferred, consistent with `add`).
  - Resolves `homonto/agents/<source>.md`; for each declared target it
    re-materializes per the agent's mode (`copy` writes the current source
    content; `link` ensures the symlink points at the source).
  - **Backup-before-overwrite**: if a `copy`-mode target's on-disk content differs
    from the recorded hash (a local edit), the current file is first copied to
    `<path>.bak` before the source content is written — no user edit is silently
    lost. (Three-way-merge is deferred.)
  - **Idempotent**: a target already matching the source (copy content-equal /
    link pointing at source) is a no-op ("up to date").
  - Updates the lockfile with the new content hash per target and reports each
    target's outcome (`updated` / `updated (backed up …)` / `up to date`).
- A newly-declared target (added to `targets` since install) is installed by
  `update` too (it re-materializes every declared target). De-declared targets are
  left in place (reported by `doctor`; pruning is a later concern).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents update`, which re-materializes an
  installed `local:` agent from its source (backing up locally-modified copies)
  and refreshes the lockfile — the fix action for the drift `agents doctor`
  reports.

## Impact

- `internal/cli/agents.go`: new `update` subcommand (`agentsUpdateCmd`), reusing
  the `add` install helpers (`isSymlinkTo`, `link.Link`, `fsutil.WriteAtomic`,
  `subagentpath.Dir`) and `agentlock`.
- Tests in `internal/cli`.
- No new dependency. `list`/`add`/`doctor` and all prior behavior unchanged.
- Deferred: three-way-merge (vs backup); builtin/remote sources; de-declared-
  target pruning; `migrate`; per-agent scope.
