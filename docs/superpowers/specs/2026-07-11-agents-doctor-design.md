---
comet_change: agents-doctor
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-agents-doctor
status: final
---


v2 #3. After `agents add` created the `.homonto/agents-lock.json` installed-state
ground truth, `agents doctor` compares it against the config (declared) and disk.
Read-only; the peer of `homonto doctor`/`onto doctor` for agents. Reuses
`agentlock.Load`/`HashContent`. Findings + non-zero exit, mirroring `onto doctor`.

## Goals / Non-Goals

**Goals**: `homonto agents doctor` — read-only drift report (declared-vs-installed
-vs-disk), `healthy`+0 or findings+non-zero.

**Non-Goals**: fixing drift (that's `update`/`add`/`migrate`); builtin/remote
source hashing; three-way-merge; touching plan/apply/state.json.

## Decisions

### D1 — `agentsDoctorCmd` (`internal/cli/agents.go`)

```
cfgPath := --config; cfgDir := filepath.Dir(cfgPath); homontoDir := cfgDir/.homonto
c := config.Load(cfgPath)          // declared
lock := agentlock.Load(homontoDir) // installed
findings := nil  // []string, keyed by agent name for readability

// 1. declared agents
for name, ag := range c.Agents (sorted):
    inst, installed := lock.Agents[name]
    if !installed: finding "<name>: declared but not installed (run `homonto agents add <name>`)"; continue
    // source drift (local: only)
    if strings.HasPrefix(ag.Source,"local:"):
        srcPath := cfgDir/homonto/agents/<trimprefix>.md
        b, err := os.ReadFile(srcPath)
        if err: finding "<name>: source file <srcPath> missing or unreadable"
        else if any recorded target hash != HashContent(b): finding "<name>: source changed since install (re-run `homonto agents add <name>`)"
    // declared targets
    declaredTargets := ag.TargetsOrAll()
    for tool in declaredTargets (sorted):
        ti, ok := inst.Installed[tool]
        if !ok: finding "<name>: target <tool> declared but not installed"; continue
        if _, err := os.Lstat(ti.Path); err != nil: finding "<name> (<tool>): installed file missing: <ti.Path>"; continue
        if inst.Mode == "copy":
            b, err := os.ReadFile(ti.Path)
            if err: finding "... unreadable"
            else if HashContent(b) != ti.Hash: finding "<name> (<tool>): modified on disk: <ti.Path>"
        // link mode: on-disk file present is sufficient this increment
    // installed target no longer declared
    for tool in inst.Installed (sorted):
        if tool not in declaredTargets: finding "<name>: target <tool> installed but no longer targeted"

// 2. orphans: installed agents not declared
for name in lock.Agents (sorted):
    if _, ok := c.Agents[name]; !ok: finding "<name>: installed but no longer declared (orphan)"

// verdict
if len(findings)==0: cmd.Println("healthy"); return nil
for f in findings: cmd.Println(f)
return fmt.Errorf("homonto agents doctor: %d problem(s) found", len(findings))
```
Register `doctor` under `agentsCmd()`. Root `SilenceErrors/SilenceUsage` + main's
`error: <err>` → clean non-zero exit (same as `onto doctor`).

### D2 — Source-hash comparison

An install records the same content hash for every target (all materialized from
one source). So "source drifted" = current `homonto/agents/<x>.md` hash differs
from ANY recorded target hash (they're equal at install time). Compare against
the first recorded target's hash (deterministic: sort or just any). Use
`agentlock.HashContent` — identical to what `add` recorded, so no false drift.

### D3 — Deterministic output

All iterations sorted by name/tool so findings order is stable (Go map order is
random). Mirrors `onto doctor`.

## Risks / Trade-offs

- **Source drift vs on-disk drift are distinct findings**: source drift means the
  provider file changed (re-add needed); modified-on-disk means the installed copy
  was edited (local-edit — a future `update` decides merge/backup). Both reported;
  a later increment adds the fix action.
- **link mode**: this increment only checks the link's target file exists (via
  Lstat on the recorded path). Verifying the symlink still points at the source is
  a nice-to-have deferred (add records the path, not the link target separately).
- **Read-only**: no writes; findings use recorded/derived paths only.

## Migration Plan

Additive read-only command. No migration.

## Open Questions

None. The fix actions (`update`/`migrate`) that consume this drift are deferred,
scoped follow-ups.
