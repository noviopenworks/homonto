# Consolidate Claude/OpenCode copy-mode projection onto a shared copyproj core

## Why

Roadmap F40 copy-mode follow-on (the structured-doc and file-projection slices
already landed via `consolidate-structured-doc-projection` and
`consolidate-file-projection`). The `claude` and `opencode` adapters carry a
byte-for-byte identical copy-mode subagent reconciler — `recordedCopyHashes`,
`copySubagentName`, `planCopyOps`, `applyCopySubagents` — differing only in the
literal tool string (`"claude"`/`"opencode"`) and the conflict error prefix.
`internal/copyfile` already owns the reconcile primitive; the adapter-level
orchestration around it is the last duplicated adapter surface.

## What Changes

Add `internal/adapter/copyproj` and route both adapters' `subagentcopy.*`
handling through it, completing the adapter-consolidation story.

- `copyproj` owns: `Name(dst)` (subagent name from a managed copy-file dst),
  `Plan(tool, desired, st)` (recorded-hash lookup + `copyfile.Plan`), and
  `Apply(tool, desired, st, pruneRoots)` (the local-edit `.bak` promotion +
  `copyfile.Apply` + state recording, with the conflict error keyed by `tool`).
  The `subagentcopy.` state prefix and `.md` suffix live here (both adapters
  share them).
- Each adapter keeps only `copySubagentDesired()` (builds the desired
  dst→content map from its own subagent dirs) and `copyPruneRoots()` (its own
  managed roots), then calls `copyproj`.

## Impact

- **Specs:** `adapter-contract` gains a requirement that the built-in adapters
  reconcile copy-mode content files through the shared core.
- **Behavior:** none — pure refactor pinned by the conformance suite plus the
  per-adapter copy-mode tests. The F7 prune-root guard and local-edit backup
  behavior are preserved exactly.
- **Risk:** low — the smallest of the F40 slices, a mechanical extraction of
  identical logic, guarded by existing tests.

## Non-goals

- Three-way merge for copy-mode local edits (still backup+overwrite; a separate
  future item).
- Any change to `internal/copyfile` semantics or any adapter behavior/schema.
