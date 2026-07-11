# v2 #5 — Agent Three-Way Merge / `migrate` — Design (for review)

> **Umbrella design — superseded by the implementing archive chain.** This is the
> shared pre-implementation design for the v2 #5 three-way-merge work. It was
> implemented across three archived changes:
> [`2026-07-11-agents-merge-core`](technical-design.md) (#5a, the merge/blob
> foundation), `openspec/changes/archive/2026-07-11-agents-merge-update` (#5b,
> wiring merge into `update`), and
> `openspec/changes/archive/2026-07-11-agents-update-all` (`update --all`). It
> lives here (rather than in a single change) because it spans that chain; the
> per-change `technical-design.md` files carry the as-built decisions.

Pre-implementation design (roadmap v2 Agent Lifecycle, increment #5). Presented
for approval before any code. Covers the four forks the user flagged: base-content
storage, merge algorithm, `migrate` vs `update` semantics, and conflict UX.

## Problem

Today `agents update` re-materializes an installed `local:` agent from source and,
for a locally-modified `copy` install, **backs up** the local file to `<dst>.bak`
then **overwrites** with the new source (clobber+backup — the user's edit is
preserved but discarded from the live file). #5 replaces clobber+backup with a
real **three-way merge**: reconcile the user's local edits with the upstream
source change automatically when they don't overlap, and surface a conflict only
when they do.

Three-way merge needs three inputs:
- **BASE** — the content that was last installed (the common ancestor).
- **LOCAL** — the current on-disk file (may contain the user's edits).
- **UPSTREAM** — the current source content (`homonto/agents/<x>.md`).

The lockfile (`internal/agentlock`) records per-target `{Path, Hash}` — a hash,
**not** the base content. So the base is unavailable today. Fork 1 solves that.

## Fork 1 — Base-content storage  →  **Recommend: content-addressed blob store**

| Option | Shape | Trade-off |
|---|---|---|
| A. Inline in lockfile | base content as a JSON string field per target | bloats `agents-lock.json` with multi-line markdown, ugly escaping, large diffs |
| **B. Blob store (recommended)** | `.homonto/agents-blobs/<sha256>` files; lockfile keeps the existing `Hash` as the key | small lockfile; content-addressed dedup across targets; base retrievable via `blobs/<prev.Hash>` |

**B needs almost no schema change**: the lockfile already stores each install's
content `Hash`. On `add`/`update`, additionally write the installed content to
`.homonto/agents-blobs/<hash>` (idempotent — skip if the blob exists). Then on a
later `update`, `BASE = read(blobs/<prev.Hash>)`. A tiny `internal/agentblob`
package (`Put(dir, content) -> hash`, `Get(dir, hash) -> content, ok`).

- **GC**: unreferenced blobs accumulate as agents change. Defer a `blobs` GC (a
  later `agents doctor --gc` or archive step); note it, don't build it in #5.
- **Missing base blob** (e.g. lockfile hand-edited, or a pre-#5 install with no
  blob): fall back to today's backup+overwrite behavior for that target, and
  note it. So #5 degrades gracefully on installs made before blobs existed.

## Fork 2 — Merge algorithm  →  **Recommend: minimal line-based diff3, own `internal/merge` package**

| Option | Behavior | Trade-off |
|---|---|---|
| A. Equality-only + markers | if LOCAL==BASE → take UPSTREAM; if UPSTREAM==BASE → keep LOCAL; if both changed → conflict | trivial, but never auto-merges non-overlapping edits (worse than git) |
| **B. Line-based diff3 (recommended)** | diff LOCAL vs BASE and UPSTREAM vs BASE by line; auto-merge non-overlapping hunks; emit conflict markers on overlap | real value; ~150–250 LOC; no stdlib diff3 in Go |

**B**: implement a small, well-tested `internal/merge` package:
`func Merge(base, local, upstream []byte) (result []byte, conflicts int)`.
Approach: compute a line-level LCS/diff of `local` vs `base` and `upstream` vs
`base` (a compact Myers or Hunt–Szymanski LCS on lines — ~80 LOC), walk both
edit scripts together, and:
- a region changed on only one side → take that side;
- a region unchanged on both → take base;
- a region changed on both, identically → take it once;
- a region changed on both, differently → emit conflict markers
  (`<<<<<<< local` / `=======` / `>>>>>>> source`), `conflicts++`.

No third-party dependency (keeps homonto's small-surface philosophy). The package
is pure and unit-testable in isolation (dozens of cases: no-op, one-side, both-
non-overlap, both-overlap, adjacent, EOF-newline handling) before any CLI wiring.

Rejected: vendoring a diff3 dep — avoidable, and a bounded in-repo implementation
is more auditable and dependency-free.

## Fork 3 — `migrate` vs `update`  →  **Recommend: merge folds into `update`; `migrate` is a thin "update-all"**

In a declarative model there is no separate "migration" state to step through —
the config is the target and `update` reconciles one agent toward it. So:
- **`update <name>`** GAINS the merge: BASE (blob) + LOCAL (disk) + UPSTREAM
  (source) → 3-way-merge. Clean merge → write result, refresh base blob + lock.
  Conflict → Fork-4 UX. When there is no local edit (LOCAL==BASE) it degrades to
  today's plain refresh; when no source change (UPSTREAM==BASE) it is a no-op.
- **`migrate`** (optional, later): a convenience that runs `update` across **all**
  installed agents and summarizes clean-vs-conflicted — not a distinct algorithm.
  Recommend deferring `migrate` to #5c (or dropping it: `update <name>` per agent
  plus a future `update --all` covers the need). The roadmap lists `migrate`, but
  its real content is "update everything + report" — thin over `update`.

## Fork 4 — Conflict UX  →  **Recommend: safe-by-default `.merged` sidecar, live file untouched**

| Option | On conflict | Trade-off |
|---|---|---|
| A. git-style markers in the live file | write merged+markers to `<dst>`, back up local to `<dst>.orig`, exit non-zero | familiar, but the **live agent file is left broken** (markers) until resolved — the tool loads a broken agent |
| **B. sidecar (recommended)** | leave `<dst>` (LOCAL) **untouched**; write the merged-with-markers result to `<dst>.merged`; report the conflict + exit non-zero; `doctor` reports "conflicted (resolve <dst>.merged)" | live file never broken; user reviews `.merged`, copies the resolution over `<dst>`, re-runs `update` (now LOCAL==resolved, clean) |

**B** fits homonto's "never break the user's working state" stance: a conflicted
`update` changes nothing live, writes a reviewable `.merged`, and exits non-zero.
Clean (non-conflicting) merges DO write `<dst>` (with a `.bak` of the prior local,
same as update today) and refresh the base blob. `doctor` gains a "conflicted"
finding when a `<dst>.merged` exists / the lockfile marks it. (Alternative A can
be offered later behind a `--markers` flag if users want the git flow.)

## Proposed implementation slices

1. **#5a — `internal/merge`** (pure diff3 package) + **`internal/agentblob`**
   (`.homonto/agents-blobs/<sha256>` Put/Get), each fully unit-tested. `add`/
   `update` start writing the base blob on install (no behavior change yet).
2. **#5b — wire merge into `update`**: BASE (blob) + LOCAL + UPSTREAM → merge;
   clean → write + `.bak` + refresh blob/lock; conflict → `.merged` sidecar +
   non-zero + report; missing-base-blob → today's backup+overwrite fallback.
   `doctor` reports the conflicted state.
3. **#5c (optional) — `agents migrate` / `update --all`**: run the merge across all
   installed agents, summarize clean vs conflicted. Defer or drop.

Each slice is one verified comet cycle (open→design→build→verify→archive).

## Non-goals for #5

Remote sources (#6), builtin-source hashing, `[agents]`-vs-`[subagents]`
reconciliation, blob GC, per-agent scope, semantic/AST merge (line-based only).

## Resolved decisions (approved 2026-07-11)

1. **Fork 4 conflict UX**: **safe `.merged` sidecar** — the live `<dst>` is left
   untouched; the merged-with-markers result goes to `<dst>.merged`; `update`
   exits non-zero and `doctor` reports "conflicted". (git-style in-file markers
   may be added later behind a `--markers` flag — not in #5.)
2. **`migrate`**: implement **`agents update --all`** (slice #5c) that merges every
   installed agent and summarizes clean-vs-conflicted; `migrate` is a thin alias/
   wrapper over it.
3. **Merge granularity**: **line-based diff3** — sufficient for markdown agents; no
   word/char-level.
