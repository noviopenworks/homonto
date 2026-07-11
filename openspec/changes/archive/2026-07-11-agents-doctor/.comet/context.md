# Comet Design Handoff

- Change: agents-doctor
- Phase: design
- Mode: compact
- Context hash: 729f937534ef029fc109e79109765cc58086a705be73c5391dfc277fa18dcdde

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-doctor/proposal.md

- Source: openspec/changes/agents-doctor/proposal.md
- Lines: 1-55
- SHA256: 44ff5a0f504b99bad06b2d9ea963df2d9ec4d013e33d5b6ffbb246858d818ed1

```md
## Why

v2 added the `[agents.<name>]` model, `agents list`, and `agents add` (with the
`.homonto/agents-lock.json` lockfile of what's installed). The next lifecycle
piece is *health*: comparing what's declared (config) against what's installed
(lockfile + disk) so a user knows an agent is missing, orphaned, or drifted
before `update`/`migrate` land. This change adds `homonto agents doctor` — a
read-only diagnostic, the peer of `homonto doctor`/`onto doctor` for the agent
lifecycle. It also unblocks later increments (`update`/`migrate` act on the drift
`doctor` reports).

## What Changes

- Add `homonto agents doctor`: read-only, loads the config (declared agents) and
  `.homonto/agents-lock.json` (installed), and reports each problem as a finding:
  - **declared but not installed** — a declared agent with no lockfile record
    (run `agents add`);
  - **orphaned** — a lockfile-recorded agent no longer declared in config;
  - **source drifted** — a `local:` agent whose `homonto/agents/<x>.md` content
    hash no longer matches the recorded install hash (re-run `agents add`), or
    whose source file is now missing;
  - **target not installed** — a target the agent declares that has no lockfile
    install entry (e.g. a newly added target);
  - **target no longer declared** — an installed target the agent no longer
    targets;
  - **missing on disk** — a recorded install path that no longer exists;
  - **modified on disk** — a `copy`-mode install whose on-disk content hash no
    longer matches the recorded hash.
  On a healthy workspace it prints `healthy` and exits 0; with findings it prints
  each and exits non-zero (CI/scriptable, like `onto doctor`). It writes nothing.
- Register `doctor` under `agentsCmd()`. `homonto agents` now has list / add /
  doctor.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents doctor`, a read-only health check
  reporting declared-vs-installed drift (missing/orphaned/source-drifted/
  target-mismatch/missing-on-disk/modified-on-disk), non-zero exit on findings.

## Impact

- `internal/cli/agents.go`: new `doctor` subcommand (`agentsDoctorCmd`).
- Reuses `internal/agentlock` (`Load`, `HashContent`) and `internal/subagentpath`
  (only for path context if needed — findings use recorded paths).
- Tests in `internal/cli`.
- No new dependency. Read-only; no projection/mutation. All prior behavior
  unchanged.
- Deferred: `update`/`pin`/`migrate` (which act on this drift), builtin/remote
  sources, three-way-merge.

```

## openspec/changes/agents-doctor/design.md

- Source: openspec/changes/agents-doctor/design.md
- Lines: 1-94
- SHA256: 0fe2d4341f1288912159925dd70071b1b27d2b37b3d39e26614a0458fa42d8c8

[TRUNCATED]

```md
## Context

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

```

Full source: openspec/changes/agents-doctor/design.md

## openspec/changes/agents-doctor/tasks.md

- Source: openspec/changes/agents-doctor/tasks.md
- Lines: 1-11
- SHA256: e76d510413f49b57c58f6b32e3578c5fd59ff803a14093489ef6e10947a84b8a

```md
## 1. `homonto agents doctor` (`internal/cli`)

- [ ] 1.1 (TDD RED first) `agentsDoctorCmd` (`doctor`, NoArgs) per Design Doc D1/D2/D3: load config + `agentlock.Load`; accumulate findings (sorted iteration) — declared-not-installed; local: source drift (source missing / hash != recorded) via `agentlock.HashContent`; per declared target: not-installed / missing-on-disk (Lstat recorded path) / copy modified-on-disk (ReadFile hash != recorded); installed-target-no-longer-declared; orphan (installed not declared). Verdict: 0 findings → `healthy` + nil; else print each + return `fmt.Errorf("homonto agents doctor: %d problem(s) found", n)`. Register `doctor` under `agentsCmd()`.
- [ ] 1.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","doctor","--config",p])` in a temp workspace (config + homonto/agents + .homonto/agents-lock.json seeded, or built by running `agents add` first): healthy → nil err, stdout `healthy`; declared-not-installed → non-nil naming agent; orphan → non-nil; source drift (edit source after add) → non-nil; modified-on-disk (edit installed copy) → non-nil; missing-on-disk (delete installed file) → non-nil; read-only (no files created). Prefer building state by invoking `agents add` in-test for realism.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents doctor' reports declared-vs-installed drift`

## 2. Regression and docs

- [ ] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add a local agent, `agents doctor` → `healthy` exit 0; edit the source file → `agents doctor` reports drift exit non-zero; delete an installed file → reports missing-on-disk.
- [ ] 2.2 Update `docs/roadmap.md` v2 status (agents doctor landed) + README (mention `homonto agents doctor`). No over-claim.
- [ ] 2.3 Commit all changes.

```

## openspec/changes/agents-doctor/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-doctor/specs/agent-lifecycle/spec.md
- Lines: 1-64
- SHA256: 5fe35f27cee4bf45367041e6d42319a936f10c682974f1a6aac97e0125e0f77e

```md
## ADDED Requirements

### Requirement: homonto agents doctor reports agent health

`homonto agents doctor` SHALL be a read-only command that loads the config
(declared agents) and `.homonto/agents-lock.json` (installed agents) and reports
each drift as a finding. It SHALL write nothing. It SHALL check:

- a declared agent absent from the lockfile is **not installed**;
- a lockfile-recorded agent absent from the config is **orphaned**;
- a `local:` agent whose `homonto/agents/<source>.md` content hash differs from
  the recorded install hash (or whose source file is missing) has a **source
  drift**;
- a target the agent declares but has no lockfile install entry is a **target not
  installed**;
- a lockfile install entry for a target the agent no longer declares is a
  **target no longer declared**;
- a recorded install path that no longer exists on disk is **missing on disk**;
- a `copy`-mode install whose on-disk content hash differs from the recorded hash
  is **modified on disk**.

On a healthy workspace it SHALL print `healthy` and exit 0. When one or more
findings exist it SHALL print each finding and exit non-zero.

#### Scenario: healthy workspace

- **GIVEN** a config whose declared agents are all installed, undrifted, and unmodified per the lockfile and disk
- **WHEN** `homonto agents doctor` runs
- **THEN** it prints `healthy` and exits 0

#### Scenario: declared but not installed

- **GIVEN** a `[agents.<name>]` with no lockfile record
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the agent as not installed and exits non-zero

#### Scenario: orphaned install

- **GIVEN** a lockfile agent that is no longer declared in the config
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the agent as orphaned and exits non-zero

#### Scenario: source drift

- **GIVEN** an installed `local:` agent whose source file content changed since install
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the source drift and exits non-zero

#### Scenario: modified on disk

- **GIVEN** a copy-mode installed agent whose on-disk file content was edited
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the file as modified on disk and exits non-zero

#### Scenario: missing on disk

- **GIVEN** a recorded install whose file was deleted
- **WHEN** `homonto agents doctor` runs
- **THEN** it reports the file as missing on disk and exits non-zero

#### Scenario: read-only

- **WHEN** `homonto agents doctor` runs
- **THEN** it writes no files and mutates nothing

```
