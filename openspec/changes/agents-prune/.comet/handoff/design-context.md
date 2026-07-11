# Comet Design Handoff

- Change: agents-prune
- Phase: design
- Mode: compact
- Context hash: e62d2d9449f31d489d71dc33400446fe0209fd3b947a936663c6e246588371d3

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-prune/proposal.md

- Source: openspec/changes/agents-prune/proposal.md
- Lines: 1-50
- SHA256: bfe2e7b9563f7e092143bf019d659be9ddead6068b9936256931041146a9c30c

```md
## Why

`agents doctor` reports two kinds of stale installs — an **orphan** (an agent in
the lockfile no longer declared in the config) and a **de-declared target** (a
target recorded for an agent that the agent no longer targets) — but nothing
removes them. This change adds `homonto agents prune` to clean them up safely,
completing the lifecycle: `add` installs, `doctor` detects drift, `prune` removes
what you removed from the config.

## What Changes

- Add `homonto agents prune`: removes homonto-managed agent installs that are no
  longer declared, and drops their lockfile records.
  - **Orphan agent** (in `.homonto/agents-lock.json`, not in the config): each of
    its recorded target install files is removed and the agent's lockfile entry
    is dropped.
  - **De-declared target** (a target in an agent's `Installed` that the agent no
    longer targets): that target's install file is removed and its `Installed`
    entry dropped; the agent (and its still-declared targets) is kept.
  - **Safety**: only a file at a homonto-*recorded* install path is touched. A
    file whose on-disk content differs from the recorded base hash (a local edit)
    is backed up to `<path>.bak` before removal — no user edit is silently lost.
    A pruned target's leftover `<path>.merged` conflict sidecar is also removed.
  - It reports each pruned item (and any backup), saves the lockfile, and prints
    `nothing to prune` when the workspace is already clean.
- `--dry-run` lists what would be pruned without changing anything.
- Register `prune` under `agentsCmd()`. `homonto agents` gains add / doctor /
  list / prune / update.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents prune`, which removes homonto-managed
  installs for orphaned agents and de-declared targets (backing up any
  locally-modified file first) and drops their lockfile records; `--dry-run`
  previews.

## Impact

- `internal/cli/agents.go`: new `prune` subcommand (`agentsPruneCmd`), reusing
  `agentlock` and the existing sorted/hash helpers.
- Tests in `internal/cli`.
- No new dependency. `add`/`list`/`doctor`/`update` unchanged.
- Deferred: blob GC (unreferenced `.homonto/agents-blobs/*`) — a separate
  concern; prune does not GC blobs (they may be shared / content-addressed).

```

## openspec/changes/agents-prune/design.md

- Source: openspec/changes/agents-prune/design.md
- Lines: 1-95
- SHA256: 1eda4fc3d5fa3661359b065abcf6a7f95cd3bbacb3b8968d7132c5a1dca2aa53

[TRUNCATED]

```md
## Context

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


```

Full source: openspec/changes/agents-prune/design.md

## openspec/changes/agents-prune/tasks.md

- Source: openspec/changes/agents-prune/tasks.md
- Lines: 1-11
- SHA256: 1e81ce05ee4b6d6064716f9cc272508382536230a70c8be0c174a547d3aef7fd

```md
## 1. `homonto agents prune` (`internal/cli`)

- [ ] 1.1 (TDD RED first) `agentsPruneCmd` (`prune`, NoArgs, `--dry-run` bool) per Design Doc D1/D2/D3: load config + lockfile; for each lockfile agent — orphan (not declared) → prune every recorded target file + drop `lock.Agents[name]`; else de-declared target (recorded target not in TargetsOrAll) → prune that file + drop from Installed. `pruneFile`: skip if missing; dry-run→record "would remove"; else back up to `.bak` when on-disk hash != recorded base hash, remove file, remove `.merged` sidecar. Report actions; `nothing to prune` when none; `--dry-run` changes nothing (no Save). Save once when changed. Register `prune` under `agentsCmd()`.
- [ ] 1.2 (TDD RED first) Tests (build via `agents add`, then de-declare in a new config): orphan agent → its file(s) removed + lockfile entry gone; de-declared target (agent keeps another target) → only that target's file removed + Installed entry gone, agent stays; local-edit orphan → `.bak` created with the edit before removal; `.merged` sidecar of a pruned target removed; nothing-to-prune → message, no changes; `--dry-run` → lists prunable but removes nothing and lockfile unchanged. Assert on disk + lockfile.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents prune' removes orphaned/de-declared agent installs`

## 2. Regression and docs

- [ ] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add an agent; remove it from config; `agents doctor` reports orphan; `agents prune --dry-run` lists it; `agents prune` removes the file + lockfile entry; `agents doctor` → healthy. A de-declared target similarly pruned.
- [ ] 2.2 Update `docs/roadmap.md` v2 status (agents prune landed) + README (mention `agents prune`). No over-claim.
- [ ] 2.3 Commit all changes.

```

## openspec/changes/agents-prune/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-prune/specs/agent-lifecycle/spec.md
- Lines: 1-50
- SHA256: 516e2cdf74ef956351389ff12d6de82a929b1bd4c8b2e91aca079252b274393b

```md
## ADDED Requirements

### Requirement: homonto agents prune removes stale managed installs

`homonto agents prune` SHALL remove homonto-managed agent installs that are no
longer declared and drop their lockfile records. It SHALL handle:

- an **orphan agent** (recorded in `.homonto/agents-lock.json` but not declared in
  the config): remove each recorded target install file and drop the agent's
  lockfile entry;
- a **de-declared target** (a target in an agent's `Installed` no longer in the
  agent's declared targets): remove that target's install file and drop its
  `Installed` entry, keeping the agent and its still-declared targets.

It SHALL only touch a file at a homonto-recorded install path. Before removing a
file whose on-disk content differs from the recorded base hash (a local edit), it
SHALL back the file up to `<path>.bak`. It SHALL also remove a pruned target's
leftover `<path>.merged` sidecar. It SHALL report each pruned item and print
`nothing to prune` when there is nothing to remove. A `--dry-run` flag SHALL list
what would be pruned and change nothing.

#### Scenario: prune an orphaned agent

- **GIVEN** an agent recorded in the lockfile that is no longer declared in the config
- **WHEN** `homonto agents prune` runs
- **THEN** its recorded install files are removed and its lockfile entry is dropped

#### Scenario: prune a de-declared target

- **GIVEN** an agent whose lockfile records a target the agent no longer declares
- **WHEN** `homonto agents prune` runs
- **THEN** that target's install file is removed and its `Installed` entry dropped, while the agent and its still-declared targets remain

#### Scenario: prune backs up a locally-modified install

- **GIVEN** an orphan agent whose install file was locally edited (differs from the recorded base hash)
- **WHEN** `homonto agents prune` runs
- **THEN** the file is copied to `<path>.bak` before being removed

#### Scenario: nothing to prune

- **GIVEN** a workspace whose lockfile matches the config exactly
- **WHEN** `homonto agents prune` runs
- **THEN** it reports nothing to prune and changes nothing

#### Scenario: dry run changes nothing

- **GIVEN** an orphan agent
- **WHEN** `homonto agents prune --dry-run` runs
- **THEN** it lists the orphan as prunable but removes no files and does not change the lockfile

```
