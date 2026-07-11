# Comet Design Handoff

- Change: agents-merge-core
- Phase: design
- Mode: compact
- Context hash: 9d3d5a29b876c04f20632cde801995c0f6332d9e48f571052735035eb169ee37

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-merge-core/proposal.md

- Source: openspec/changes/agents-merge-core/proposal.md
- Lines: 1-53
- SHA256: bdbe65aedcf8449c28a14d77dbeba2d3d2dc8894be9b67b0e309af9c8015502d

```md
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

```

## openspec/changes/agents-merge-core/design.md

- Source: openspec/changes/agents-merge-core/design.md
- Lines: 1-94
- SHA256: a61371be8b0d5d52ed890d258f7e358bbdc30cb711695d68927079c3d82ddf7d

[TRUNCATED]

```md
## Context

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

```

Full source: openspec/changes/agents-merge-core/design.md

## openspec/changes/agents-merge-core/tasks.md

- Source: openspec/changes/agents-merge-core/tasks.md
- Lines: 1-21
- SHA256: 2757d8b58e2a2ffc7154847145d6eb6de88e88c5236833826d3f81bf883c9f6c

```md
## 1. `internal/merge` — line-based three-way merge

- [ ] 1.1 (TDD RED first) `merge.Merge(base, local, upstream []byte) (result []byte, conflicts int)` per Design Doc D1: line split w/ exact trailing-newline round-trip; `lcsLineIndices` DP LCS; anchor intersection + sentinels; per-gap 4-way rule; git-style markers (`<<<<<<< local` / `=======` / `>>>>>>> source`).
- [ ] 1.2 (TDD RED first) Exhaustive tests: `Merge(x,x,x)==x`&0; `Merge(b,l,b)==l`&0; `Merge(b,b,u)==u`&0; disjoint edits auto-merge&0; overlapping→conflicts≥1 + both markers present; identical edits both sides→0 & single copy; adjacent edits; empty/one-line inputs; no-trailing-newline round-trip; conflict block byte-shape.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(merge): pure line-based three-way merge engine`

## 2. `internal/agentblob` — content-addressed base store

- [ ] 2.1 (TDD RED first) `Put(homontoDir, content) (hash, err)` (idempotent, sha256hex == agentlock.HashContent, WriteAtomic) + `Get(homontoDir, hash) (content, ok, err)` at `.homonto/agents-blobs/<hash>`. Tests: Put→Get round-trip; Put idempotent (same hash, single file); Get missing→(nil,false,nil); hash matches agentlock.HashContent.
- [ ] 2.2 GREEN; gofmt/vet clean. Commit: `feat(agentblob): content-addressed base-content blob store`

## 3. Persist base blobs in add/update (behavior-preserving)

- [ ] 3.1 (TDD RED first) In `agentsAddCmd`/`agentsUpdateCmd`, after materializing, `agentblob.Put(homontoDir, sourceContent)` (once per agent; propagate error). No output/flow change. Tests: after `agents add`, `.homonto/agents-blobs/<recorded hash>` exists and Get returns the source content; after `agents update` to a new source, the new source's blob exists; existing add/update/doctor tests still pass (behavior unchanged).
- [ ] 3.2 GREEN; gofmt/vet clean. Commit: `feat(cli): persist installed base content to the agent blob store`

## 4. Regression and docs

- [ ] 4.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): `agents add` a local agent → `.homonto/agents-blobs/<hash>` holds the source content; no user-visible behavior change to add/update/doctor.
- [ ] 4.2 Update `docs/roadmap.md` v2 status (merge engine + base blob store landed; #5b wires merge into update next). No over-claim (merge not yet wired into update).
- [ ] 4.3 Commit all changes.

```

## openspec/changes/agents-merge-core/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-merge-core/specs/agent-lifecycle/spec.md
- Lines: 1-62
- SHA256: 0bc90ce036855efc1b53946170367b61c12059ee3f8d56a9cc3dd85d08a6f971

```md
## ADDED Requirements

### Requirement: Three-way merge engine

The repository SHALL provide a pure, dependency-free line-based three-way merge:
`merge.Merge(base, local, upstream []byte) (result []byte, conflicts int)`. It
SHALL auto-merge changes that `local` and `upstream` make to disjoint regions of
`base`, and SHALL emit git-style conflict markers (`<<<<<<< local`, `=======`,
`>>>>>>> source`) for regions both sides changed differently, returning the count
of conflict regions. When a side is unchanged relative to base, the other side's
content SHALL be taken; when both sides made the identical change, it SHALL be
taken once.

#### Scenario: no changes

- **WHEN** `Merge(x, x, x)` is called
- **THEN** it returns `x` with 0 conflicts

#### Scenario: only local changed

- **GIVEN** `local` differs from `base` and `upstream == base`
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** it returns `local` with 0 conflicts

#### Scenario: only upstream changed

- **GIVEN** `upstream` differs from `base` and `local == base`
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** it returns `upstream` with 0 conflicts

#### Scenario: non-overlapping changes auto-merge

- **GIVEN** `local` edits an early region and `upstream` edits a later, disjoint region of `base`
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** it returns a result containing both edits with 0 conflicts

#### Scenario: overlapping changes conflict

- **GIVEN** `local` and `upstream` change the same region differently
- **WHEN** `Merge(base, local, upstream)` is called
- **THEN** the result contains a conflict-marked region and the conflict count is ≥ 1

### Requirement: Agent base-content blob store

Install operations SHALL persist installed agent content to a content-addressed
store `.homonto/agents-blobs/<sha256>`. `agentblob.Put(homontoDir, content)` SHALL
write the blob idempotently and return its sha256 hex (matching the lockfile
`Hash`); `agentblob.Get(homontoDir, hash)` SHALL read it back. `homonto agents
add` and `homonto agents update` SHALL `Put` each installed target's content, so
the base content is retrievable by the recorded install hash. This SHALL NOT
change the user-visible behavior of `add`/`update`.

#### Scenario: install persists a retrievable base blob

- **GIVEN** a local agent installed via `homonto agents add`
- **WHEN** the install completes
- **THEN** `.homonto/agents-blobs/<hash>` exists for each target's recorded hash and `agentblob.Get` returns the installed content

#### Scenario: blob Put is idempotent and content-addressed

- **WHEN** `agentblob.Put` is called twice with the same content
- **THEN** both return the same hash and the store holds a single blob

```
