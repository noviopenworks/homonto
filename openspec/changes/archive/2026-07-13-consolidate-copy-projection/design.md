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
