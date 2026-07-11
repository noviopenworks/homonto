## Why

v2 #5 replaces `agents update`'s clobber+backup with a real three-way merge (per
the approved `2026-07-11-agents-3way-merge-design.md`). This foundation slice
(#5a) adds the two dependency-free building blocks — a pure line-based diff3
merge engine and a content-addressed base-content blob store — and starts
persisting the base content on install, WITHOUT changing any user-visible
behavior yet. #5b then wires the merge into `update`.

## What Changes

- Add `internal/merge`: a pure, dependency-free line-based three-way merge.
  `func Merge(base, local, upstream []byte) (result []byte, conflicts int)`.
  Non-overlapping edits from `local` and `upstream` (relative to `base`) are
  auto-merged; overlapping edits emit git-style conflict markers
  (`<<<<<<< local` / `=======` / `>>>>>>> source`) and increment `conflicts`.
  Algorithm (per the design): line-level LCS of (base,local) and (base,upstream);
  the base lines common to BOTH LCSs are anchors; between consecutive anchors,
  apply — local-slice==base-slice → take upstream; upstream-slice==base-slice →
  take local; local==upstream → take it; else conflict.
- Add `internal/agentblob`: a content-addressed store at
  `.homonto/agents-blobs/<sha256>`. `Put(homontoDir, content) (hash string, err)`
  writes the blob idempotently (skip if present) and returns its sha256 hex
  (identical to `agentlock.HashContent`). `Get(homontoDir, hash) (content []byte,
  ok bool, err error)` reads it back.
- `agents add` and `agents update` additionally `agentblob.Put` the installed
  content after materializing each target, so the base (last-installed) content
  is retrievable by its recorded lockfile `Hash`. This is the only wiring change
  and is behavior-preserving (no output/flow change; just persists blobs).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: install operations (`add`/`update`) now persist the
  installed base content to a content-addressed blob store
  (`.homonto/agents-blobs/<sha256>`), enabling a future three-way merge on update.

## Impact

- New `internal/merge` package (pure; exhaustively unit-tested).
- New `internal/agentblob` package (blob Put/Get; unit-tested).
- `internal/cli/agents.go`: `add`/`update` call `agentblob.Put` after each
  materialize (behavior-preserving).
- Tests in `internal/merge`, `internal/agentblob`, `internal/cli`.
- No new dependency. No user-visible behavior change (the merge is wired into
  `update` in #5b).
- Deferred: #5b (merge into `update` + `.merged` sidecar + `doctor` conflicted),
  #5c (`update --all`), blob GC.
