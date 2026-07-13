---
comet_change: consolidate-copy-projection
role: technical-design
canonical_spec: openspec
status: draft
---

# consolidate-copy-projection — Technical Design

Deep design for F40's copy-mode follow-on. OpenSpec is the canonical spec; this
is the technical design and defers to `openspec/changes/
consolidate-copy-projection/specs` for normative requirements; the full API,
per-adapter split, and identity notes are in the change's `design.md`.

## Decision

Add `internal/adapter/copyproj` — the copy-mode analogue of `structproj`/
`fileproj` — wrapping the already-shared `internal/copyfile`. The two adapters'
copy-mode reconciler is byte-identical except the tool string and the conflict
error prefix, so it lifts cleanly with `tool` as a parameter.

## Why its own package (not folded into fileproj)

Copy-mode is a different primitive from symlink projection: content-hash
ownership (not "dst -> src"), local-edit `.bak` promotion, and the F7
prune-root guard. Folding it into `fileproj.Link` would muddy both. A small
dedicated package keeps each contract coherent.

## Risk posture

Lowest of the F40 slices — a mechanical extraction of identical logic, pinned by
the conformance suite and per-adapter copy-mode tests. The F7 prune-root guard
(refused prunes retain ownership, never delete out-of-root) and the local-edit
backup behavior are preserved verbatim.

## Out of scope

Three-way merge for local edits (still backup+overwrite); any `internal/copyfile`
semantic change; any adapter behavior/schema change.
