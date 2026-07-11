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
