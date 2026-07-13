# Comet Design Handoff

- Change: consolidate-copy-projection
- Phase: design
- Mode: compact
- Context hash: 66903938adb7ffe5619a2f73857ba20cda86e8e611d7c318740c944770dcd483

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/consolidate-copy-projection/proposal.md

- Source: openspec/changes/consolidate-copy-projection/proposal.md
- Lines: 1-43
- SHA256: 85c4700f6a24fd351f792bdf37e0e214c63972e181e2d205ae0422961b1a8fe0

```md
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

```

## openspec/changes/consolidate-copy-projection/design.md

- Source: openspec/changes/consolidate-copy-projection/design.md
- Lines: 1-63
- SHA256: 187c89a0580b752b1c6b1ee9afc34e15e17e226b5d419df8f9f9c3af6e9aec73

```md
# Design — consolidate copy-mode projection

## High-level approach

Add `internal/adapter/copyproj` (the copy-mode analogue of `structproj`/
`fileproj`), wrapping the already-shared `internal/copyfile`. The copy-mode
logic is byte-identical between the two adapters except the tool string and the
conflict error prefix, so it lifts cleanly with `tool` as a parameter.

### Contract API

```go
const keyPrefix = "subagentcopy."

// Name recovers the subagent name from a managed copy-file dst.
func Name(dst string) string  // TrimSuffix(Base(dst), ".md")

// Plan computes the reconciler ops for the desired copy files against state.
func Plan(tool string, desired map[string][]byte, st *state.State) ([]copyfile.Op, error)

// Apply reconciles the copy files: writes created/updated, prunes de-declared,
// backs up any local edit to <dst>.bak before overwrite/prune, records/deletes
// subagentcopy.* state. A foreign file or symlink at a dst is a conflict keyed
// by tool. pruneRoots bound where a prune may delete (F7).
func Apply(tool string, desired map[string][]byte, st *state.State, pruneRoots []string) error
```

`recordedCopyHashes` (already `tool`-parameterized) becomes an internal helper.

### Adapter side (all that remains)

Each adapter keeps `copySubagentDesired() (map[string][]byte, error)` (builds
dst→content from its `subagents`/`subagentsDir`/`subagentSource`) and
`copyPruneRoots() []string` (its user/project subagent dirs). Plan emits
`subagentcopy.*` changes via `copyproj.Plan(tool, desired, st)`; Apply calls
`copyproj.Apply(tool, desired, st, a.copyPruneRoots())`. The Plan change-emit
loop maps `op.Dst` → key via `copyproj.Name`.

## Identity-preservation notes

- Conflict error message must stay `"<tool>: <dst> exists and is not a
  homonto-managed copy-mode subagent; not overwriting"`.
- Local-edit promotion: `LocalEdit` with nil Content → `Prune` (de-declared +
  edited, backed up); with Content → `Update` (declared + edited, backed up).
- Refused prunes (dst outside pruneRoots — tampered state) are NOT in `pruned`,
  so their ownership record is retained and the out-of-root file never deleted
  (F7). Preserve exactly.
- State recording: `st.Set(tool, keyPrefix+Name(dst), dst, hash)` for each
  reconciled file; `st.Delete(tool, keyPrefix+Name(dst))` for each pruned.

## Migration order (each step green before next)

0. Add `internal/adapter/copyproj` (Name/Plan/Apply + internal recordedCopyHashes
   + keyPrefix) with table-driven tests. Green in isolation.
1. claude: replace the 4 shared helpers with copyproj calls; keep
   copySubagentDesired + copyPruneRoots. claude + conformance suites green.
2. opencode: same. opencode + conformance suites green.

## Alternatives considered

- **Fold into fileproj** — rejected; copy-mode is a different primitive
  (content-hash ownership, `.bak` promotion, prune-root guard) and would muddy
  the symlink-oriented `fileproj.Link` API. Its own small package is cleaner.

```

## openspec/changes/consolidate-copy-projection/tasks.md

- Source: openspec/changes/consolidate-copy-projection/tasks.md
- Lines: 1-18
- SHA256: 2b989e8499d26df5b14b5de2b829de33709c9d856933da884e7fbca23cf12199

```md
# Tasks — consolidate-copy-projection

## 1. copyproj core
- [ ] Add `internal/adapter/copyproj` (Name, Plan, Apply + internal
      recordedCopyHashes + keyPrefix "subagentcopy."). Table-driven tests
      (create/update/prune/local-edit/conflict/prune-root-refusal). Green.

## 2. claude migration
- [ ] Route claude copy-mode through copyproj; keep copySubagentDesired +
      copyPruneRoots; Plan emit uses copyproj.Name. claude + conformance green.

## 3. opencode migration
- [ ] Route opencode copy-mode through copyproj (same). opencode + conformance
      green.

## 4. Verify
- [ ] internal/copyfile untouched; F7 prune-root guard + local-edit backup
      preserved. `go test ./... -race`, vet, build, `openspec validate --all`.

```

## openspec/changes/consolidate-copy-projection/specs/adapter-contract/spec.md

- Source: openspec/changes/consolidate-copy-projection/specs/adapter-contract/spec.md
- Lines: 1-30
- SHA256: 1988706287260d9fbb1ac8cb27b716f7eec3c78b4ca5921c16a170f3fc396399

```md
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

```
