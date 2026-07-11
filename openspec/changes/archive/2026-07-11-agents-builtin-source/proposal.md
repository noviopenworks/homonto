## Why

The v2 agent lifecycle (`agents add / list / doctor / update [--all]` with
three-way merge) works only for `local:` sources — `builtin:` returns "not yet
supported". This change resolves `builtin:<name>` agents from the embedded
catalog (the same curated agent files the framework ships, indexed as subagents),
so a user can declare and manage a bundled agent without authoring it under
`homonto/agents/`. Remote sources remain deferred (an explicit first-release
non-goal).

## What Changes

- Add `catalog.Catalog.SubagentContent(name string) ([]byte, bool, error)`: reads
  a builtin agent's content from the embedded catalog by name (the curated agent
  files are the framework's subagents), returning `ok=false` for an unknown name.
- Add a source resolver in `internal/cli` shared by `add`/`update`/`doctor`:
  `resolveAgentSource(ag, cfgDir) ([]byte, error)` →
  - `local:<x>` → `homonto/agents/<x>.md` (as today);
  - `builtin:<x>` → the embedded catalog content (unknown → clear error);
  - anything else → "unsupported source (remote not yet supported)".
- `agents add` and `agents update` resolve the source via that resolver, so both
  now accept `builtin:` agents. All downstream logic (hashing, materialize,
  base-blob store, three-way merge, `.merged` conflict sidecar) is source-agnostic
  and works unchanged — including auto-merging a user's local edits with a
  *catalog upgrade* to a builtin agent.
- `agents doctor` resolves the source via the resolver too, so a `builtin:` agent
  gets the same "source changed since install" drift detection (a catalog upgrade
  that changes the builtin content), and an unknown/unresolvable source is a
  finding.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: `agents add`/`update`/`doctor` resolve `builtin:<name>`
  sources from the embedded catalog (in addition to `local:`), so bundled agents
  are declarable and lifecycle-managed (install, drift, three-way merge). Remote
  sources remain unsupported (deferred).

## Impact

- `internal/catalog/catalog.go`: new `SubagentContent` method.
- `internal/cli/agents.go`: `resolveAgentSource` helper; `add`/`update`/`doctor`
  use it (replacing the `local:`-only reads and the "not yet supported" branch).
- Tests in `internal/catalog` and `internal/cli` (a fixture catalog with a builtin
  agent).
- No new dependency. `local:` behavior unchanged; remote still rejected.
- Deferred: remote sources (first-release non-goal); catalog-version-aware pinning
  for builtin agents; `[agents]`-vs-`[subagents]` reconciliation.
