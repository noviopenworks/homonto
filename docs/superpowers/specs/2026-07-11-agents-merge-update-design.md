---
comet_change: agents-merge-update
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-agents-merge-update
status: final
---


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

## Risks / Trade-offs

- **doctor contract change**: modified-on-disk was a #3 finding; dropping it is
  correct for the merge model (local edits are supported), but update the #3 tests
  that asserted it. Documented; the delta spec MODIFIES the doctor requirement.
- **conflict leaves clean targets advanced but conflicted target on prev**: a
  re-run after the user resolves `.merged` re-merges the conflicted target; clean
  targets are already up-to-date (no-op). Predictable and safe.
- **`.merged` accumulation**: a stale `.merged` from a resolved conflict lingers
  until the user deletes it; doctor keeps reporting "conflicted" until then — which
  is the intended nudge. (A future `update` could clear `.merged` on a clean
  re-merge — nice-to-have, note it.)

## Migration Plan

Additive to update/doctor behavior. No data migration.

## Open Questions

None — approved design. #5c adds `update --all`.
