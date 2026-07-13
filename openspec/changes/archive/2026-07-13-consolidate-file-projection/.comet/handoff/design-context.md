# Comet Design Handoff

- Change: consolidate-file-projection
- Phase: design
- Mode: compact
- Context hash: e82e53f2285782f5416111bfd238f1c2a1833101bf5457b607d67bc8f3dee3f8

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/consolidate-file-projection/proposal.md

- Source: openspec/changes/consolidate-file-projection/proposal.md
- Lines: 1-56
- SHA256: a6dddbec0145ce00f7b9c48a0bc0a7dd62f77ccb45806be6b9abde3c780aba36

```md
# Consolidate Claude/OpenCode file-projection onto a shared fileproj contract

## Why

Roadmap F40, file-projection slice (the structured-document slice already
landed via `consolidate-structured-doc-projection`). The `claude` and
`opencode` adapters each re-implement the **symlink projection** for three
resource types — skills, commands, subagents — six near-identical copies of
security-sensitive link/relocate/adopt/prune orchestration across `Plan`,
`Apply`, and `ObserveHashes`. `internal/link` already provides the primitives
(`Plan`/`Link`/`Remove`/`IsManaged`); what is duplicated is the adapter-level
control flow around them. This mirrors the structured-doc problem `structproj`
solved, so file-projection deserves the same treatment: one shared contract.

## What Changes

Add `internal/adapter/fileproj`, the symlink analogue of `structproj`, and
route both adapters' `skill.*`/`command.*`/`subagent.*` projection through it.

- The contract is **type-agnostic**: the adapter supplies a flat
  `[]fileproj.Link{Dst, Src, Key, Inactive}` per namespace (precomputing the
  destination, content source, state key, and the same-named other-scope path).
  `fileproj` never needs to know about directories, `.md` suffixes, or scopes —
  those collapse into the precomputed fields.
- `fileproj` owns: `Project` (create/relocate/relink + adopt-unrecorded
  planning), `Conflicts` (fail-fast precheck), `ApplyState` (adopt/delete state
  side), `ApplyLinks` (inactive-prune + link + record), and `Observe` (drift
  re-hash from the recorded dst).
- Each adapter keeps only ~12 lines per type to build its `[]Link`, plus its
  existing dir/scope helpers.
- The generic delete loop stays unchanged; `fileproj` plans **no** deletes
  (file-prefix deletes remain owned solely by the generic loop — no
  double-delete).

Copy-mode subagents (`subagentcopy.*`, real files via `internal/copyfile`) are
**out of scope** — a different primitive (content-hash ownership, local-edit
promotion, prune-root guard); it gets its own follow-up.

## Impact

- **Specs:** `adapter-contract` gains a requirement that the built-in adapters
  project their managed symlinks through the shared file-projection core.
- **Behavior:** none — pure refactor pinned by `internal/adapter/conformance`
  plus the per-adapter `scope`/`adopt`/`observehashes`/`pruning`/`robustness`
  link tests. Plan/apply/observe output must be byte/behavior identical.
- **Risk:** higher than the structured-doc slice — symlinks, relocation, and
  fail-fast conflict ordering. Mitigated by: skills-first canary (the only
  directory/suffix-less case), one adapter+type at a time green before the
  next, and preserving the exact Apply phase ordering (state adopt/delete →
  all conflict prechecks → doc writes → link creation).

## Non-goals

- Copy-mode (`subagentcopy.*`) consolidation.
- Any change to `internal/link`/`internal/copyfile` semantics.
- Any adapter behavior, schema, or CLI change.

```

## openspec/changes/consolidate-file-projection/design.md

- Source: openspec/changes/consolidate-file-projection/design.md
- Lines: 1-95
- SHA256: d2bfde38810e6f3aa3cdb68d7d24de3358b71e66337be2f33bd3ed705580013c

[TRUNCATED]

```md
# Design — consolidate file-projection

## High-level approach

Add `internal/adapter/fileproj` (the symlink analogue of `structproj`) and
migrate both adapters' three link namespaces onto it. Type-agnostic contract:
the adapter precomputes a flat `[]fileproj.Link` so the shared code never
branches on resource type.

### Contract API

```go
type Link struct { Dst, Src, Key, Inactive string }

func Project(tool string, links []Link, st *state.State, roots []string) ([]adapter.Change, error)
func Conflicts(links []Link, roots []string) error
func ApplyState(tool string, changes []adapter.Change, st *state.State, roots []string, fallbackDst func(key string) string) error
func ApplyLinks(tool string, links []Link, st *state.State, roots []string) error
func Observe(tool, prefix string, st *state.State) map[string]string
```

- **Project**: `link.Plan` over `{Dst:Src}`; per op — `Cur=="" &&
  Inactive!="" && IsManaged(Inactive)` → relocate (`update`, Old=Inactive,
  New="dst -> src"); `Cur==""` → `create` (New="dst -> src"); else → `update`
  (Old=op.Cur, **New=bare src**). Then adopt-unrecorded: readlink Dst, if it
  equals Src and state's Applied != Hash("dst -> src"), emit `adopt`. Emits **no
  deletes**. Does not sort (adapter's final sort handles it).
- **Conflicts**: `link.Plan` error-only precheck.
- **ApplyState**: for prefix-filtered changes — `adopt` → `st.Set(tool,key,
  New,Hash(New))`; `delete` → resolve dst (recordedDst else fallbackDst) →
  `link.Remove` + `st.Delete`.
- **ApplyLinks**: per link — if `Inactive!="" && IsManaged(Inactive)` →
  `link.Remove(Inactive)`; then `link.Link(Src,Dst)`; then `st.Set(tool,key,
  "dst -> src",Hash("dst -> src"))`.
- **Observe**: per recorded key of prefix — read `recordedDst(e.Desired)`,
  `os.Readlink`, on error continue, else `out[key]=Hash(dst+" -> "+target)`.

`recordedDst` and the `" -> "` separator move into `fileproj` (deleted from both
adapters' util.go).

### Adapter side (the only per-type code left, ~12 lines ×3)

Each adapter builds `skillFileLinks()/commandFileLinks()/subagentFileLinks()`
computing `Dst=filepath.Join(dir(scope), name[.md])`, `Src=source(entry)`,
`Key=prefix+name`, `Inactive= inactiveDir(scope)=="" ? "" :
filepath.Join(inactiveDir(scope), filepath.Base(Dst))`. The subagent builder
skips copy-mode entries. `fallbackDst` closures mirror the current user-scope
fallback.

Plan: three `Project` calls replace the six inline blocks. Apply file section,
**order preserved**: `ApplyState`×3 → `Conflicts`×3 → copy conflict precheck
(unchanged) → JSON writes (unchanged) → `ApplyLinks`×3 → `applyCopySubagents`
(unchanged). Observe: three `fileproj.Observe` calls; `subagentcopy.` stays
inline.

## Identity-preservation risks (from design investigation)

1. **Inactive `""` sentinel** must survive — never `filepath.Join("", name)`.
2. **`filepath.Base(dst)` unification** — `Join(inactive, Base(dst))` equals the
   current `name`(skills)/`name+".md"` forms; the skills (directory) case is the
   canary to confirm.
3. **relink asymmetry** — create/relocate emit New="dst -> src"; plain relink
   emits bare New=src with Old=op.Cur. Preserve exactly.
4. **hash string** — adopt/link/observe all hash `"dst -> src"` with the exact
   `" -> "` separator recordedDst cuts on. One shared constant.
5. **fail-fast conflict ordering** (biggest risk) — all `Conflicts` prechecks
   run BEFORE any JSON write or link creation; `ApplyState` (adopt/delete) runs
   before `Conflicts`, as today. `ApplyLinks` must never be the first place a
   conflict surfaces.
6. **inactive-prune keeps the `IsManaged` guard** so `link.Remove` only touches
   our own symlink.
7. **generic delete loop stays; fileproj plans no deletes** — no double-delete.
8. **Observe reads recorded dst, not current scope** — a pending scope switch
   leaves the applied link at the old location; continue on Readlink error.
9. **ApplyState per-prefix interleaving** is harmless (state is a keyed map).
10. **Old redaction** — relocate Old is a plain path (unredacted); only the
    generic-loop delete redacts. File keys never carry secrets.

## Migration order (each step green before the next)


```

Full source: openspec/changes/consolidate-file-projection/design.md

## openspec/changes/consolidate-file-projection/tasks.md

- Source: openspec/changes/consolidate-file-projection/tasks.md
- Lines: 1-23
- SHA256: b659e3bb7852c959fea479b24e665bf8892a0d4b1b4c6d90a90a767a1374da30

```md
# Tasks — consolidate-file-projection

## 1. fileproj contract
- [ ] Add `internal/adapter/fileproj` (Link, Project, Conflicts, ApplyState,
      ApplyLinks, Observe + recordedDst + " -> " constant). Table-driven unit
      tests; green in isolation.

## 2. claude migration (skills canary → commands → subagents)
- [ ] claude skills via fileproj (narrow inline adopt/delete loop to
      command./subagent.). scope/adopt/observehashes/pruning/conformance green.
- [ ] claude commands via fileproj. Suites green.
- [ ] claude subagents via fileproj; delete now-empty inline loop + dead
      recordedDst. Suites green.

## 3. opencode migration (same sequence)
- [ ] opencode skills → commands → subagents via fileproj; drop dead
      recordedDst. opencode + conformance suites green.

## 4. Verify + scope confirm
- [ ] Copy-mode (subagentcopy.*) and internal/link untouched; generic delete
      loop unchanged (fileproj plans no deletes).
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      byte/behavior identical (conformance + per-adapter link tests).

```

## openspec/changes/consolidate-file-projection/specs/adapter-contract/spec.md

- Source: openspec/changes/consolidate-file-projection/specs/adapter-contract/spec.md
- Lines: 1-42
- SHA256: 16dc7660b3022f0cdf02f0010d25162fe594794381eed5e83f3599edc40d18d9

```md
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

```
