---
comet_change: agents-update
role: technical-design
canonical_spec: openspec
---


v2 #4. `agents doctor` detects drift; `agents update` fixes it by re-materializing
from source. Declarative model → no `pin` command (version is config). Backup (not
merge) protects local edits this increment. Reuses the `add` install helpers.

## Goals / Non-Goals

**Goals**: `homonto agents update <name>` re-installs a declared+installed
`local:` agent from source, backing up locally-modified copies, idempotent,
lockfile refreshed.

**Non-Goals**: three-way-merge (backup only); builtin/remote sources; pruning
de-declared targets; `migrate`; installing a not-yet-installed agent (that's
`add`); per-agent scope.

## Decisions

### D1 — `agentsUpdateCmd` (`internal/cli/agents.go`)

Same setup as `add` (cfgDir/homontoDir, config.Load, agentlock.Load, home). Then:
```
ag, ok := c.Agents[name]; if !ok -> error "agent %q is not declared"
if !strings.HasPrefix(ag.Source,"local:") -> "only local: sources supported yet"
inst, installed := lock.Agents[name]; if !installed -> error "agent %q is not installed (run `homonto agents add %s`)"
srcName := trimprefix; srcPath := cfgDir/homonto/agents/<srcName>.md
content, err := os.ReadFile(srcPath); if err -> error naming srcPath
hash := agentlock.HashContent(content)
installedRec := map[string]agentlock.Install{}
for tool in ag.TargetsOrAll() (sorted):
    dir := subagentpath.Dir(tool,"user",home,""); dst := dir/name+".md"
    prev, hadRec := inst.Installed[tool]
    switch ag.ModeOrDefault():
    case "copy":
        cur, statErr := os.ReadFile(dst)
        if statErr == nil && agentlock.HashContent(cur) == hash:
            status "up to date"   // already matches source
        else:
            if statErr == nil && hadRec && agentlock.HashContent(cur) != prev.Hash:
                // locally modified vs last install → back up before overwrite
                fsutil.WriteAtomic(dst+".bak", cur); note backup
            mkdirall(dir); fsutil.WriteAtomic(dst, content); status "updated"[+backup]
    case "link":
        if isSymlinkTo(dst, srcPath): status "up to date"
        else: link.Link(srcPath, dst, homontoDir); status "updated"
    installedRec[tool] = {Path:dst, Hash:hash}
lock.Agents[name] = {Source,Version,Mode:ModeOrDefault,Targets:TargetsOrAll,Installed:installedRec}
lock.Save(homontoDir)
print per-target status
```
Register `update` under `agentsCmd()`.

### D2 — Backup semantics

Backup fires ONLY for copy mode when the on-disk content differs from BOTH the
source (else it's already up to date / a plain refresh) AND the last recorded
install hash (i.e. a genuine LOCAL edit, not just a stale copy of an older
source). `<path>.bak` is a plain copy via `fsutil.WriteAtomic` (overwrites a prior
`.bak`; one level of backup is the contract this increment). No backup for link
mode (the file is a symlink; re-pointing loses nothing). No backup when the target
is simply missing (nothing to preserve).

Refinement: distinguish "source changed, install still equals OLD source"
(on-disk == prev.Hash) — that is NOT a local edit, so NO backup, just overwrite.
Only on-disk != prev.Hash AND on-disk != new hash → local edit → backup. When
on-disk == prev.Hash (untouched since install) → overwrite silently.

### D3 — Reuse add's helpers

`isSymlinkTo`, `link.Link`, `fsutil.WriteAtomic`, `subagentpath.Dir`, `agentlock`
are all already imported by agents.go — no new deps.

## Risks / Trade-offs

- **Backup vs merge**: backup is lossless and simple; a user can diff `.bak`
  against the new file. Three-way-merge (auto-reconcile local + upstream changes)
  is a deferred #5. Documented.
- **One-level `.bak`**: a second update overwrites the prior `.bak`. Acceptable
  for this increment; a timestamped/rotated backup is a later refinement.
- **update ≠ add**: update refuses an uninstalled agent (points to `add`),
  keeping the two operations distinct and predictable.

## Migration Plan

Additive. No migration.

## Open Questions

None. Three-way-merge and de-declared-target pruning are deferred, scoped.
