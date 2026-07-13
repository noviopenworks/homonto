# adapter-contract

## ADDED Requirements

### Requirement: Built-in adapters reconcile copy-mode content files through the shared core

The `claude` and `opencode` adapters SHALL reconcile their copy-mode content
files (`subagentcopy.*`) through the shared `internal/adapter/copyproj` core —
planning create/update/prune ops, backing up a local edit to `<dst>.bak` before
overwrite or prune, recording and deleting `subagentcopy.*` state, and refusing
to delete a prune destination that resolves outside the adapter's managed roots
— rather than each re-implementing that orchestration. Each adapter supplies
only the desired destination→content map and its managed prune roots. The
reconcile behavior, including the conflict abort and the prune-root guard, MUST
be identical to the prior per-adapter implementation, as pinned by the shared
conformance suite and the per-adapter copy-mode tests.

#### Scenario: Adapters reconcile copy-mode files through the core

- **WHEN** an adapter plans and applies its copy-mode subagent content files
- **THEN** it does so through `copyproj.Plan` and `copyproj.Apply`, and the
  resulting ops, on-disk files, backups, and recorded state are identical to the
  prior per-adapter implementation

#### Scenario: Prune-root guard is preserved

- **WHEN** a `subagentcopy.*` state entry names a prune destination outside the
  adapter's managed roots
- **THEN** the shared core refuses to delete it and retains its ownership record,
  never deleting an out-of-root file
