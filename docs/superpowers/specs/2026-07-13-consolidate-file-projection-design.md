---
comet_change: consolidate-file-projection
role: technical-design
canonical_spec: openspec
status: draft
---

# consolidate-file-projection — Technical Design

Deep design for F40 (file-projection slice). OpenSpec is the canonical spec;
this is the technical design and defers to `openspec/changes/
consolidate-file-projection/specs` for normative requirements. The full
contract API, adapter-side builders, the ten identity-preservation risks, and
the skills-first migration order are specified in the change's `design.md`;
this document records the architecture decision and its rationale.

## Decision

Introduce `internal/adapter/fileproj` — the symlink analogue of the
already-shipped `internal/adapter/structproj` — and route both JSON adapters'
`skill.*` / `command.*` / `subagent.*` projection through it.

## Why type-agnostic

The three link namespaces vary in only four knobs (destination dir, inactive
dir, `.md` suffix, key prefix). Crucially `filepath.Join(inactive,
filepath.Base(dst))` equals every type's current inactive-orphan path form, so
if the adapter precomputes each link's `{Dst, Src, Key, Inactive}`, the shared
core needs **zero** knowledge of dirs/suffixes/scopes. This collapses six
near-identical adapter blocks into one contract + ~12 lines of per-type builder.

## Why fileproj plans no deletes

Unlike `structproj` (whose keys share a JSON document, so it owns their
deletes and they are excluded from the generic loop), file-prefix deletes are
emitted solely by each adapter's generic delete loop (`filePrefix` /
`managedPrefix`). Keeping that the single delete source avoids a double-delete;
`fileproj` only *consumes* the deletes the generic loop produces (in
`ApplyState`).

## Why copy-mode is out

`subagentcopy.*` is a different primitive: real files via `internal/copyfile`,
content-hash ownership, local-edit promotion (`.bak`), and the F7 prune-root
guard. Folding it into the `Link` API would muddy both. It gets its own
follow-up wrapping the already-shared `internal/copyfile`.

## Risk posture

Higher than the structured-doc slice (symlinks, relocation, fail-fast conflict
ordering). Mitigations: skills-first canary (the only directory/suffix-less
case — if it reproduces identically, the `.md` types are guaranteed); one
adapter+type at a time, `conformance` + per-adapter link tests green before the
next; the exact Apply phase ordering (state adopt/delete → all conflict
prechecks → doc writes → link creation) preserved verbatim.

## Out of scope

Copy-mode consolidation; any `internal/link`/`internal/copyfile` semantic
change; any adapter behavior/schema/CLI change.
