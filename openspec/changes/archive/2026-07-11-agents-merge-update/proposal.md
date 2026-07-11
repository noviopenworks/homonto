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
