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
