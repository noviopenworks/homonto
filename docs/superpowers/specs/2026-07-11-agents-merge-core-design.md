---
comet_change: agents-merge-core
role: technical-design
canonical_spec: openspec
---


v2 #5a — the dependency-free foundation for three-way-merge on `update` (approved
design: docs/superpowers/specs/2026-07-11-agents-3way-merge-design.md). Two pure
building blocks + behavior-preserving blob persistence. #5b wires merge into
`update`.

## Goals / Non-Goals

**Goals**: `internal/merge` (line diff3), `internal/agentblob` (content-addressed
base store), `add`/`update` persist base blobs (no behavior change).

**Non-Goals**: wiring merge into `update` (#5b); `.merged` sidecar / doctor
conflicted (#5b); `update --all` (#5c); blob GC; word/char-level merge.

## Decisions

### D1 — `internal/merge.Merge(base, local, upstream []byte) (result []byte, conflicts int)`

Line-based, deterministic, pure. Algorithm (correct + tractable; over-conflicts
only in rare cases — always safe, never mis-merges):

1. Split each input into lines, preserving whether the input ended with a
   trailing newline (join back the same way). Use `strings.SplitAfter(s,"\n")`-
   style or split on "\n" and remember the trailing-newline flag; be consistent
   so `Merge(x,x,x)==x` byte-exactly.
2. `commonL := lcsLineIndices(base, local)` → the base line indices kept in local
   (a strictly increasing []int), plus their matched local indices.
   `commonU := lcsLineIndices(base, upstream)` similarly. Use a straightforward
   O(n·m) dynamic-programming LCS on line equality (agent files are small).
3. `anchors` := base indices present in BOTH commonL and commonU (intersection,
   still increasing) — lines unchanged in all three. For each anchor keep its
   matched local index and upstream index.
4. Add sentinel anchors: a virtual start (baseIdx -1, localIdx -1, upstreamIdx -1)
   and end (baseIdx len(base), localIdx len(local), upstreamIdx len(upstream)).
   Walk consecutive anchor pairs (p, q); for the gap:
   - `B := base[p.b+1 : q.b]`, `L := local[p.l+1 : q.l]`, `U := upstream[p.u+1 : q.u]`.
   - if equalLines(L, B): emit U (local unchanged in this gap → take upstream)
   - else if equalLines(U, B): emit L (upstream unchanged → take local)
   - else if equalLines(L, U): emit L (identical change both sides)
   - else: emit conflict block `<<<<<<< local\n`+L+`=======\n`+U+`>>>>>>> source\n`,
     conflicts++
   - then emit the anchor line q (unless q is the end sentinel).
5. Join emitted lines back to []byte with the original trailing-newline behavior.

Helpers: `lcsLineIndices` (DP LCS returning aligned index pairs), `equalLines`
(slice equality). Keep marker strings exact: `<<<<<<< local`, `=======`,
`>>>>>>> source` each on their own line.

Property tests (must): `Merge(x,x,x)==x` & 0; `Merge(b,l,b)==l` & 0;
`Merge(b,b,u)==u` & 0; disjoint edits merge & 0; overlapping edits → conflicts≥1
and result contains both markers; adjacent edits; empty inputs; no-trailing-
newline inputs; identical edits both sides → 0 conflicts, single copy.

### D2 — `internal/agentblob`

`.homonto/agents-blobs/<sha256hex>` files.
- `Put(homontoDir string, content []byte) (hash string, err error)`: `hash :=
  sha256hex(content)` (same as `agentlock.HashContent`); path :=
  `homontoDir/agents-blobs/<hash>`; if it exists → return hash (idempotent); else
  `fsutil.WriteAtomic(path, content)`; return hash.
- `Get(homontoDir, hash string) (content []byte, ok bool, err error)`: read the
  file; missing → (nil,false,nil); other error → (nil,false,err).
- Reuse `agentlock.HashContent` (or a shared sha256 helper) so blob hash == lock
  hash exactly — the lockfile `Hash` IS the blob key.

### D3 — Wire base-blob persistence into add/update (behavior-preserving)

In `agentsAddCmd` and `agentsUpdateCmd`, after each target is materialized (or
found up-to-date), call `agentblob.Put(homontoDir, content)` where `content` is
the SOURCE content just installed (the same bytes whose hash is recorded). This
persists the base for a future merge. It changes no output and no control flow;
a `Put` error should propagate (rare — disk). Only for `copy`... and `link`? For
link mode the "installed content" is the source file's content — Put it too (the
base for a future link-mode consideration; cheap). Simplest: Put the source
`content` once per agent (not per target) since all targets share it.

## Risks / Trade-offs

- **Over-conflict**: the anchor-intersection merge may conflict in rare cases a
  full diff3 would auto-merge. Accepted — it is always SAFE (never silently
  mis-merges); #5b's sidecar makes a conflict non-destructive anyway. Documented.
- **Blob accumulation**: no GC this slice (deferred). Blobs are small text.
- **Trailing newline**: must round-trip exactly so `Merge(x,x,x)==x` and an
  idempotent update stays idempotent — explicit test.

## Migration Plan

Additive; blobs created going forward. Pre-#5a installs simply have no blob (the
#5b merge falls back to backup+overwrite when the base blob is missing).

## Open Questions

None — approved design. #5b consumes these; #5c adds `update --all`.
