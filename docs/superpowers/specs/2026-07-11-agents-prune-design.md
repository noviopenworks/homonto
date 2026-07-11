---
comet_change: agents-prune
role: technical-design
canonical_spec: openspec
---


v2 polish — `agents prune` cleans up what `doctor` reports: orphaned agents
(installed, no longer declared) and de-declared targets. Completes the lifecycle
(add→doctor→prune). Read-mostly; touches only recorded managed paths; backs up
local edits.

## Goals / Non-Goals

**Goals**: `agents prune` (+ `--dry-run`) removes orphan-agent and de-declared-
target installs + drops lockfile records; backs up locally-modified files; removes
leftover `.merged`; reports; safe (recorded paths only).

**Non-Goals**: blob GC (blobs are content-addressed, may be shared — a separate
increment); pruning `.bak` files; touching declared/active installs.

## Decisions

### D1 — `agentsPruneCmd` (`internal/cli/agents.go`)

```
cfgPath→cfgDir→homontoDir; c := config.Load; lock := agentlock.Load
dryRun flag (bool)
var actions []string   // human-readable pruned items
changed := false
for name in sortedKeysAgents(lock.Agents):
    ag, declared := c.Agents[name]
    inst := lock.Agents[name]
    if !declared:
        // orphan: prune every recorded target
        for tool in sortedKeys(inst.Installed):
            pruneFile(inst.Installed[tool], dryRun, &actions, name, tool, "orphan")
        if !dryRun { delete(lock.Agents, name); changed = true }
        actions += "pruned orphan agent <name>"
    else:
        // de-declared targets: recorded target not in ag.TargetsOrAll()
        declaredSet := set(ag.TargetsOrAll())
        for tool in sortedKeys(inst.Installed):
            if !declaredSet[tool]:
                pruneFile(inst.Installed[tool], dryRun, &actions, name, tool, "de-declared target")
                if !dryRun { delete(inst.Installed, tool); lock.Agents[name] = inst; changed = true }
if len(actions)==0: cmd.Println("nothing to prune"); return nil
for a in actions: cmd.Println(a)
if dryRun: cmd.Println("(dry run — nothing changed)"); return nil
if changed: lock.Save(homontoDir)
```

`pruneFile(ti Install, dryRun, ...)`:
```
if _, err := os.Lstat(ti.Path); err != nil { return }  // already gone → nothing
if dryRun: actions += "would remove <ti.Path>"; return
// back up a local edit before removing
if b, err := os.ReadFile(ti.Path); err == nil && agentlock.HashContent(b) != ti.Hash {
    fsutil.WriteAtomic(ti.Path+".bak", b); actions += "backed up <ti.Path> to .bak"
}
os.Remove(ti.Path)
os.Remove(ti.Path + ".merged")   // clean up a leftover conflict sidecar (ignore error)
actions += "removed <ti.Path>"
```
Register `prune` under `agentsCmd()`.

### D2 — Safety

- Only recorded `Install.Path`s are removed — never an arbitrary/unmanaged file.
- A missing recorded file (already deleted) is a silent no-op (still drops the
  lockfile record on a real prune).
- Local-edit backup: on-disk hash != recorded base hash → `.bak` first. (For a
  merged install the on-disk may legitimately differ from base → it gets backed
  up, which is the safe choice for a file being removed.)
- `.merged` sidecar of a pruned target is removed (best-effort).
- `--dry-run` performs NO writes/removes and does not Save.

### D3 — Lockfile mutation only on real prune

On `--dry-run`, `lock` is not mutated and not saved. On a real prune, orphan →
`delete(lock.Agents, name)`; de-declared target → `delete(inst.Installed, tool)`
then reassign; Save once at the end if anything changed.

## Risks / Trade-offs

- **De-declared target of a still-declared agent**: only that target is removed;
  the agent's other targets and lockfile record persist. Predictable.
- **No blob GC**: pruning an orphan leaves its base blobs in `agents-blobs/`
  (content-addressed, possibly shared). Accepted — GC is a separate increment.
- **`.bak` accumulation**: a backed-up prune leaves a `.bak`; one level, like
  update. The user removed the agent from config, so this is a deliberate
  cleanup with a safety net.

## Migration Plan

Additive command. No migration.

## Open Questions

None.
