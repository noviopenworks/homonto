# Comet Design Handoff

- Change: agents-merge-update
- Phase: design
- Mode: compact
- Context hash: 7dfed52a760122b83156281aef8edd216a849f56d106cc086b001c3ed144daba

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-merge-update/proposal.md

- Source: openspec/changes/agents-merge-update/proposal.md
- Lines: 1-66
- SHA256: 1e4662cc7e70a829c4fa2f5a40d6270760fff94b41a030d83eaea4d219b777eb

```md
## Why

#5a built the merge engine (`internal/merge`) and the base-content blob store
(`internal/agentblob`), and `add`/`update` now persist the base. This slice (#5b)
delivers the payoff: `agents update` performs a real three-way merge of the
user's local edits with the upstream source change instead of clobbering (and
`.bak`-ing) the local file. Conflicts are surfaced safely via a `<dst>.merged`
sidecar (the approved UX) without breaking the live agent file. `doctor` is
updated for the merge world: a locally-edited install is now a normal, mergeable
state (not an error), and a pending conflict is reported.

## What Changes

- **`agents update` (copy mode) becomes a three-way merge.** For each declared
  target, with `BASE = agentblob.Get(<recorded base hash>)`, `LOCAL = on-disk`,
  `UPSTREAM = current source`:
  - up-to-date (on-disk == source) → no-op ("up to date").
  - BASE unavailable (no blob / not previously recorded / on-disk missing) →
    graceful fallback to the pre-#5b behavior (back up a genuine local edit, then
    write the source).
  - BASE available → `result, conflicts := merge.Merge(BASE, LOCAL, UPSTREAM)`:
    - **0 conflicts** → write `result` to `<dst>`; the new recorded base becomes
      `UPSTREAM` (`Install.Hash = hash(source)`, `agentblob.Put(source)`) — so the
      next update merges against the pristine source, not the merged output;
      status "merged" (or "up to date" if result == on-disk).
    - **≥1 conflict** → leave the live `<dst>` **untouched**, write the
      merged-with-markers `result` to `<dst>.merged`, do NOT change the lockfile,
      report the conflict, and exit non-zero. (No data loss: the working file is
      never broken.)
  - link mode is unchanged (re-point/refresh only).
- **`agents doctor` updated for the merge model:**
  - the `modified on disk` finding (on-disk ≠ recorded base) is **reframed**: a
    locally-edited install is a normal, mergeable state, so it is no longer a
    problem finding (it does not force a non-zero exit);
  - a new **conflicted** finding fires when a `<dst>.merged` sidecar exists
    ("conflicted (resolve <dst>.merged then re-run agents update)"), exiting
    non-zero;
  - `source changed since install` (source ≠ base) and `missing on disk` findings
    are unchanged.
- Backup: on a clean merge that changes the file, the prior local is still saved
  to `<dst>.bak` (one-level, as today) before the merged result is written.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: `homonto agents update` three-way-merges local edits with the
  upstream source (auto-merge when disjoint; a `<dst>.merged` sidecar + non-zero
  exit on conflict, live file untouched), advancing the recorded base to the
  upstream. `homonto agents doctor` treats a locally-edited install as normal and
  reports a pending merge conflict.

## Impact

- `internal/cli/agents.go`: `agentsUpdateCmd` copy-mode path rewritten to
  three-way merge (uses `merge.Merge`, `agentblob.Get/Put`); `agentsDoctorCmd`
  reframes modified-on-disk and adds the `.merged` conflicted finding.
- Tests in `internal/cli`.
- No new dependency. `add`, `list`, link-mode update, and prior behavior otherwise
  unchanged.
- Deferred: #5c `agents update --all` (migrate); git-style in-file markers behind
  a `--markers` flag; blob GC; builtin/remote sources.

```

## openspec/changes/agents-merge-update/design.md

- Source: openspec/changes/agents-merge-update/design.md
- Lines: 1-101
- SHA256: 3f1c5460bf9c05c097ad84e471c35d593660960d7a6bad4b53c53ed9de83d795

[TRUNCATED]

```md
## Context

v2 #5b — wire the #5a merge engine into `agents update` (approved design
docs/superpowers/specs/2026-07-11-agents-3way-merge-design.md). Replaces update's
clobber+backup with 3-way-merge; safe `.merged` sidecar on conflict; doctor
reframed for the merge model.

## Goals / Non-Goals

**Goals**: `update` copy-mode → 3-way merge; clean→write+advance base to upstream;
conflict→`.merged` sidecar + non-zero + live file untouched; missing-base→fallback
to today's backup+overwrite; doctor: drop modified-on-disk as a problem, add
`.merged` conflicted finding.

**Non-Goals**: `update --all` (#5c); `--markers` in-file mode; link-mode merge;
blob GC; builtin/remote.

## Decisions

### D1 — `update` copy-mode 3-way merge (`agentsUpdateCmd`, internal/cli/agents.go)

Per declared target (copy mode), `dst`, `prev, hadRec := inst.Installed[tool]`,
`content` = source bytes, `hash` = HashContent(content):
```
cur, readErr := os.ReadFile(dst)
if readErr == nil && HashContent(cur) == hash:      // on-disk already == source
    status = "up to date"; installedRec[tool] = {dst, hash}; continue-ish
else:
    base, baseOK, _ := agentblob.Get(homontoDir, prev.Hash)   // ancestor
    if readErr != nil || !hadRec || !baseOK:
        // FALLBACK (pre-#5b behavior): back up a genuine local edit, overwrite
        if readErr == nil && hadRec && HashContent(cur) != prev.Hash { WriteAtomic(dst+".bak", cur); backedUp }
        WriteAtomic(dst, content); status = "updated"[+backup]; installedRec[tool]={dst,hash}
    else:
        result, conflicts := merge.Merge(base, cur, content)
        if conflicts == 0:
            if !bytes.Equal(result, cur):                       // file changes
                if HashContent(cur) != prev.Hash { WriteAtomic(dst+".bak", cur); backedUp }  // preserve prior local
                WriteAtomic(dst, result); status = "merged"[+backup]
            else: status = "up to date"                          // merge == current on-disk
            installedRec[tool] = {dst, hash}   // NEW BASE = upstream(source) hash
            agentblob.Put(homontoDir, content) // persist new base blob (=source)
        else:
            WriteAtomic(dst+".merged", result)  // sidecar; live dst UNTOUCHED
            status = "CONFLICT (resolve " + dst + ".merged)"
            conflictErr = true                  // do NOT record installedRec[tool]; keep prev
            // keep the agent's PRIOR lockfile entry for this target unchanged
```
- Two-phase per agent: run all targets; if ANY conflicted, DON'T rewrite the
  agent's lockfile entry (leave inst as-is) and return a non-zero summary error
  after printing statuses. If none conflicted, `lock.Agents[name] = {..., Installed:
  installedRec}`, Save. (A partial merge across targets: clean targets ARE written
  to disk + blobs Put, but the lockfile entry is only committed when no target
  conflicts — matches "conflict blocks the agent's advance"; the clean targets'
  files are already correct and re-running after conflict resolution is
  idempotent.)  Simpler alternative acceptable: commit clean targets' records and
  mark conflicted ones by leaving prev; but keep it predictable — on any conflict,
  return non-zero; still Save the successfully-merged targets' new records (so a
  resolved re-run is a no-op for them). Choose: **Save merged targets, skip
  conflicted target's record (keep prev), return non-zero.**
- `agentblob.Put(content)` (new base = source) happens for every clean/merged
  target so the ancestor advances.

### D2 — doctor reframe (`agentsDoctorCmd`)

- Remove the `modified on disk` problem finding (on-disk != recorded base is now
  the normal locally-edited state). Do NOT append it.
- Add: for each installed target, if `<ti.Path>.merged` exists (os.Lstat) →
  finding `"<name> (<tool>): conflicted (resolve <ti.Path>.merged, then re-run `homonto agents update <name>`)"`.
- Keep: declared-not-installed, orphan, source-changed (source != base),
  target-not-installed / no-longer-declared, missing-on-disk.

### D3 — Base semantics (the crux)

`Install.Hash` = the recorded BASE (ancestor) = the source content last installed
or merged-against; `agentblob` stores it. After `add`: base=source, on-disk=source
(equal). After a clean `update` merge: base advances to the NEW source; on-disk =
merged (base + local edits). So on-disk != base is EXPECTED (local edits) and is
NOT drift — hence D2 drops modified-on-disk. `source changed` (current source !=
base) remains the true "an update is available" signal.

```

Full source: openspec/changes/agents-merge-update/design.md

## openspec/changes/agents-merge-update/tasks.md

- Source: openspec/changes/agents-merge-update/tasks.md
- Lines: 1-16
- SHA256: 4f3676b2af669fc3e6cc5723cc31627c8bb343eeda8f9c041dd172f5c5ed6bb3

```md
## 1. `agents update` three-way merge (`internal/cli`)

- [ ] 1.1 (TDD RED first) Rewrite `agentsUpdateCmd` copy-mode per Design Doc D1: up-to-date no-op; BASE=`agentblob.Get(prev.Hash)`; missing base/on-disk → fallback backup+overwrite; else `merge.Merge(base,cur,content)` — 0 conflicts → write result (+ `.bak` of prior local when it changes) + advance base (Install.Hash=source hash, `agentblob.Put(source)`); ≥1 conflict → write `<dst>.merged`, leave live dst + that target's lock entry unchanged, exit non-zero. Save merged targets; on any conflict return a non-zero summary error. Link mode unchanged.
- [ ] 1.2 (TDD RED first) Tests (build via add, then perturb local + source): disjoint local+source edits → dst has both, no `.merged`, base advanced (doctor healthy after); overlapping edits → live dst UNCHANGED, `<dst>.merged` has markers, exit non-zero, lockfile entry for that target unchanged; idempotent (no perturbation) → "up to date", no `.merged`/`.bak`; missing base blob (delete the blob) + local edit + source change → fallback backs up local to `.bak` and writes source; multi-target where one conflicts → clean target advanced, conflicted target sidecar + non-zero.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update' three-way-merges local edits (safe .merged sidecar)`

## 2. `agents doctor` merge-model reframe (`internal/cli`)

- [ ] 2.1 (TDD RED first) Per Design Doc D2: drop the `modified on disk` problem finding; add a `<ti.Path>.merged`-exists → "conflicted" finding; keep the rest. Update the prior #3 doctor tests that asserted modified-on-disk (a locally-edited install is now healthy). Tests: locally-edited install (source unchanged) → doctor NOT a problem (exit 0 absent other issues); a `.merged` sidecar → doctor "conflicted" + non-zero; source-changed + missing-on-disk still findings.
- [ ] 2.2 GREEN; gofmt/vet clean. Commit: `feat(cli): agents doctor reframed for merge model (local edits ok, conflicts reported)`

## 3. Regression and docs

- [ ] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add; local edit disjoint from a source edit → `agents update` merges both (doctor healthy); overlapping edits → update writes `<dst>.merged`, live file intact, exit non-zero, doctor "conflicted"; resolve (copy .merged over dst, rm .merged) + update → clean.
- [ ] 3.2 Update `docs/roadmap.md` v2 status + README (`agents update` now merges; conflicts → `.merged`). No over-claim.
- [ ] 3.3 Commit all changes.

```

## openspec/changes/agents-merge-update/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-merge-update/specs/agent-lifecycle/spec.md
- Lines: 1-76
- SHA256: 213d6901a63e4067b74926431efe0bc4fdec337e1956094af7c71e84bd7b527e

```md
## MODIFIED Requirements

### Requirement: homonto agents update re-materializes an installed agent

`homonto agents update <name>` SHALL reconcile an already-installed declared
`local:` agent with its current source. The agent MUST be declared and recorded
in the lockfile; an undeclared or not-yet-installed agent SHALL be an error (the
latter directing the user to `agents add`). `builtin:`/remote sources SHALL return
a clear "not yet supported" error.

For each declared target in `copy` mode, with `BASE` = the recorded base content
(from the blob store), `LOCAL` = the on-disk file, and `UPSTREAM` = the current
source, the command SHALL:

- no-op when the on-disk content already equals the source ("up to date");
- when the base content is unavailable (no blob recorded, or the on-disk file is
  missing), fall back to backup-before-overwrite (a genuine local edit is copied
  to `<dst>.bak` before the source is written);
- otherwise perform a three-way merge (`merge.Merge(BASE, LOCAL, UPSTREAM)`):
  - **0 conflicts** → write the merged result to `<dst>` (backing up the prior
    local to `<dst>.bak` when it changes), and advance the recorded base to
    `UPSTREAM` (so the next update merges against the pristine source);
  - **≥1 conflict** → leave the live `<dst>` unchanged, write the
    merged-with-markers result to `<dst>.merged`, make no lockfile change, report
    the conflict, and exit non-zero.

`link`-mode targets are re-pointed only (no merge). The command SHALL remain
idempotent for an already-reconciled agent.

#### Scenario: non-overlapping local + upstream edits auto-merge

- **GIVEN** an installed copy agent, a local edit to one region, and a source edit to a disjoint region
- **WHEN** `homonto agents update <name>` runs
- **THEN** `<dst>` contains both edits, no `<dst>.merged` is created, and the recorded base advances to the source

#### Scenario: overlapping edits conflict via a sidecar

- **GIVEN** an installed copy agent whose local edit and source edit overlap
- **WHEN** `homonto agents update <name>` runs
- **THEN** the live `<dst>` is unchanged, a `<dst>.merged` with conflict markers is written, the lockfile is unchanged, and the command exits non-zero

#### Scenario: update is idempotent

- **GIVEN** an installed agent already equal to its source
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target is a no-op and no `.merged`/`.bak` is created

#### Scenario: missing base blob falls back to backup

- **GIVEN** an installed copy agent with a local edit but no recorded base blob
- **WHEN** `homonto agents update <name>` runs
- **THEN** the prior local is backed up to `<dst>.bak` and the source overwrites `<dst>`

### Requirement: homonto agents doctor reports agent health

`homonto agents doctor` SHALL remain a read-only command reporting declared-vs-
installed drift with a non-zero exit on any problem finding. In the three-way-
merge model a locally-edited install (on-disk content differing from the recorded
base) is a normal, mergeable state and SHALL NOT be a problem finding. Doctor
SHALL still report: a declared-but-not-installed agent; an orphaned lockfile
agent; a `local:` source whose content differs from the recorded base ("source
changed"); a target declared-but-not-installed or installed-but-no-longer-
declared; a missing-on-disk install; and, newly, a **pending conflict** when a
`<dst>.merged` sidecar exists.

#### Scenario: locally-modified install is not a problem

- **GIVEN** an installed agent whose on-disk file was edited but whose source is unchanged
- **WHEN** `homonto agents doctor` runs
- **THEN** it does not report a problem for the local edit and (absent other issues) exits 0

#### Scenario: a pending merge conflict is reported

- **GIVEN** a `<dst>.merged` sidecar left by a conflicted `agents update`
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the target as conflicted (pointing at `<dst>.merged`) and exits non-zero

```
