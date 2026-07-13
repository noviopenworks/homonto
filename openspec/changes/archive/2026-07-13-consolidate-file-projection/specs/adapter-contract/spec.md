# adapter-contract

## ADDED Requirements

### Requirement: Built-in adapters project managed symlinks through the shared file-projection core

The `claude` and `opencode` adapters SHALL project their managed symlink
resources (`skill.*`, `command.*`, `subagent.*`) through the shared
`internal/adapter/fileproj` core — planning create/relocate/relink and
adopt-unrecorded changes, running fail-fast link-conflict prechecks before any
mutation, pruning managed inactive-scope links, creating links and recording
state, and re-hashing recorded links for drift — rather than each
re-implementing that control flow per resource type. Each adapter supplies only
a flat list of desired links (destination, content source, state key, and
same-named other-scope path). The projection behavior MUST be identical to the
prior per-type implementation, as pinned by the shared conformance suite and the
per-adapter link tests.

The file-projection core plans no deletions; de-declared managed keys are pruned
by the adapter's existing generic delete loop. Copy-mode content files
(`subagentcopy.*`) are outside this core's scope.

#### Scenario: Adapters plan and apply symlinks through the core

- **WHEN** an adapter plans, applies, and observes its `skill.*`, `command.*`,
  and `subagent.*` managed symlinks
- **THEN** it does so through `fileproj.Project` / `Conflicts` / `ApplyState` /
  `ApplyLinks` / `Observe`, and the resulting changes, on-disk links, and
  observed drift hashes are identical to the prior per-type implementation

#### Scenario: File-projection core plans no deletes

- **WHEN** a managed symlink key is no longer declared in config
- **THEN** the file-projection core does not emit a delete for it; the adapter's
  generic delete loop prunes it exactly once, preserving prior behavior

#### Scenario: Fail-fast conflict ordering is preserved

- **WHEN** applying a change set where a managed symlink destination is occupied
  by foreign content
- **THEN** the adapter detects the conflict via the core's precheck before any
  document write or link creation, leaving disk and state unmutated
