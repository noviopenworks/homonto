# Comet Design Handoff

- Change: agents-update
- Phase: design
- Mode: compact
- Context hash: 61d9278a8a046f6a86d31104c1cb9c6860e97c97aeeaca8989a7cbd6fd1c55a9

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-update/proposal.md

- Source: openspec/changes/agents-update/proposal.md
- Lines: 1-57
- SHA256: beebb8de74600ae2651834c463a29c9f9a6921740e799b0f84ca308d88d854f5

```md
## Why

`agents doctor` (v2 #3) reports when an installed agent has drifted — its
`local:` source file changed, or its installed copy was modified/deleted on disk.
The complementary *action* is missing: re-materializing the agent from its
current source so the install matches the source again. This change adds
`homonto agents update` — the fix for the drift `doctor` detects. Because homonto
is declarative (the config is the source of truth and homonto never edits
`homonto.toml`), version *pinning* is simply editing `[agents.<name>].version` in
the config, so no separate `pin` command is needed; `update` is the real lifecycle
mutation. To protect user work, a locally-modified install is **backed up** before
being overwritten (full three-way-merge is a later increment).

## What Changes

- Add `homonto agents update <name>`: re-installs an already-installed declared
  agent from its current source, refreshing `.homonto/agents-lock.json`.
  - The agent must be declared and already installed (in the lockfile); an
    uninstalled agent returns an error pointing at `agents add`. `local:` sources
    only (builtin/remote deferred, consistent with `add`).
  - Resolves `homonto/agents/<source>.md`; for each declared target it
    re-materializes per the agent's mode (`copy` writes the current source
    content; `link` ensures the symlink points at the source).
  - **Backup-before-overwrite**: if a `copy`-mode target's on-disk content differs
    from the recorded hash (a local edit), the current file is first copied to
    `<path>.bak` before the source content is written — no user edit is silently
    lost. (Three-way-merge is deferred.)
  - **Idempotent**: a target already matching the source (copy content-equal /
    link pointing at source) is a no-op ("up to date").
  - Updates the lockfile with the new content hash per target and reports each
    target's outcome (`updated` / `updated (backed up …)` / `up to date`).
- A newly-declared target (added to `targets` since install) is installed by
  `update` too (it re-materializes every declared target). De-declared targets are
  left in place (reported by `doctor`; pruning is a later concern).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents update`, which re-materializes an
  installed `local:` agent from its source (backing up locally-modified copies)
  and refreshes the lockfile — the fix action for the drift `agents doctor`
  reports.

## Impact

- `internal/cli/agents.go`: new `update` subcommand (`agentsUpdateCmd`), reusing
  the `add` install helpers (`isSymlinkTo`, `link.Link`, `fsutil.WriteAtomic`,
  `subagentpath.Dir`) and `agentlock`.
- Tests in `internal/cli`.
- No new dependency. `list`/`add`/`doctor` and all prior behavior unchanged.
- Deferred: three-way-merge (vs backup); builtin/remote sources; de-declared-
  target pruning; `migrate`; per-agent scope.

```

## openspec/changes/agents-update/design.md

- Source: openspec/changes/agents-update/design.md
- Lines: 1-89
- SHA256: 5828441bce539494bb12d1a67dcf558410765068db48519e09b34c6611c83520

[TRUNCATED]

```md
## Context

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

```

Full source: openspec/changes/agents-update/design.md

## openspec/changes/agents-update/tasks.md

- Source: openspec/changes/agents-update/tasks.md
- Lines: 1-11
- SHA256: 9bb17ac0b9de6bb0e551d175d341dcdbbebcc04a15bc8c7c51f6f4589d900e99

```md
## 1. `homonto agents update` (`internal/cli`)

- [ ] 1.1 (TDD RED first) `agentsUpdateCmd` (`update <name>`, ExactArgs(1)) per Design Doc D1/D2: setup like `add`; undeclared→err; non-local→"not yet supported"; not-installed (absent from lockfile)→err pointing to `agents add`; resolve source (missing→err); per declared target (sorted) re-materialize by mode — copy: up-to-date if on-disk hash==source hash else back up to `<dst>.bak` ONLY when on-disk hash != recorded prev.Hash AND != source hash (genuine local edit) then WriteAtomic source; link: up-to-date if isSymlinkTo(dst,src) else link.Link; record Installed{path, source-hash}; Save lock; print per-target status. Register `update` under `agentsCmd()`.
- [ ] 1.2 (TDD RED first) Tests (build state via `agents add`, then perturb): source changed → update rewrites each target to new content + lockfile hash refreshed, no .bak (install was untouched); locally-modified install (edit dst) then source also changed → update backs up dst to dst.bak (old content) and writes new source; idempotent (no perturbation) → "up to date", no .bak, no rewrite; not-installed agent → err → `agents add`; builtin → "not yet supported"; undeclared → err; link-mode update after source change → symlink still valid, "up to date" (link points at source).
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update' re-materializes an installed agent (backup-safe)`

## 2. Regression and docs

- [ ] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add copy agent; edit source → `agents update` refreshes install (doctor then healthy); edit the installed file + source → update creates `.bak` with the local edit; re-run update → "up to date".
- [ ] 2.2 Update `docs/roadmap.md` v2 status + README (mention `homonto agents update`). No over-claim (backup, not merge).
- [ ] 2.3 Commit all changes.

```

## openspec/changes/agents-update/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-update/specs/agent-lifecycle/spec.md
- Lines: 1-51
- SHA256: 9cab6fd0971b65dd702257130f006beeb713d1ecf22defd87094f7f38b337f37

```md
## ADDED Requirements

### Requirement: homonto agents update re-materializes an installed agent

`homonto agents update <name>` SHALL re-install an already-installed declared
agent from its current source and refresh `.homonto/agents-lock.json`. The agent
MUST be declared and recorded in the lockfile; an undeclared or not-yet-installed
agent SHALL be an error (the latter directing the user to `agents add`). This
increment supports `local:<x>` sources only; `builtin:`/remote sources SHALL
return a clear "not yet supported" error.

For each of the agent's declared targets the command SHALL re-materialize per the
agent's mode: `copy` writes the current `homonto/agents/<x>.md` content; `link`
ensures the symlink points at the source. It SHALL be:

- **backup-preserving**: before overwriting a `copy`-mode target whose on-disk
  content differs from the recorded hash (a local edit), the current file SHALL be
  copied to `<path>.bak`;
- **idempotent**: a target already matching the source SHALL be a no-op;
- **recorded**: the lockfile SHALL be refreshed with each target's new content
  hash.

#### Scenario: update re-materializes a changed source

- **GIVEN** an installed copy-mode `local:` agent whose source file content changed since install
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target file is rewritten to the new source content and the lockfile hash is refreshed

#### Scenario: update backs up a locally-modified install

- **GIVEN** an installed copy-mode agent whose on-disk file was edited (differs from the recorded hash)
- **WHEN** `homonto agents update <name>` runs
- **THEN** the current file is copied to `<path>.bak` before the source content overwrites it

#### Scenario: update is idempotent

- **GIVEN** an installed agent already matching its source
- **WHEN** `homonto agents update <name>` runs
- **THEN** each target is a no-op and no `.bak` is created

#### Scenario: update requires a prior install

- **GIVEN** a declared agent with no lockfile record
- **WHEN** `homonto agents update <name>` runs
- **THEN** it errors that the agent is not installed and points to `agents add`

#### Scenario: builtin source is not yet supported

- **GIVEN** an installed-or-declared `builtin:` agent
- **WHEN** `homonto agents update <name>` runs
- **THEN** it returns a clear error that builtin sources are not yet supported

```
