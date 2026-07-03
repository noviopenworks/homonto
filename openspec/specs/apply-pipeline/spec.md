# apply-pipeline Specification

## Purpose
TBD - created by archiving change homonto-v1-core. Update Purpose after archive.
## Requirements
### Requirement: Plan is a pure dry run

`homonto plan` SHALL compute and print the diff between desired and current state
without writing any file, resolving any secret, or contacting the secret backend.

#### Scenario: Plan writes nothing
- **WHEN** the user runs `homonto plan`
- **THEN** a terraform-style diff is printed and no tool file, symlink, or state
  file is created or modified

#### Scenario: Plan shows creates and updates, hides noops
- **WHEN** the plan contains create, update, and noop changes
- **THEN** the output shows `+` for creates and `~` for updates and omits noops

### Requirement: Apply is confirmation-gated and two-phase

`homonto apply` SHALL print the plan, require confirmation (`[y/N]`, skippable
with `--yes`), and then apply in two phases: resolve **all** secrets for confirmed
changes first, and only if every resolution succeeds proceed to write. State SHALL
be saved last.

#### Scenario: Confirmation declined
- **WHEN** the user answers anything other than `y`
- **THEN** apply aborts and no file is written

#### Scenario: Missing secret aborts before any write
- **WHEN** a confirmed change references a secret that cannot be resolved
- **THEN** apply aborts before writing any file, names the missing reference, and
  leaves every tool file and the state file unchanged

### Requirement: Atomic writes

Every file write SHALL go through a temp file followed by rename, so an
interrupted apply never leaves a half-written file; `state.json` SHALL be written
after all tool files.

#### Scenario: Crash-safety ordering
- **WHEN** apply writes multiple tool files and then state
- **THEN** each tool file is individually valid at all times and state is written
  only after all tool files succeed

### Requirement: Idempotent re-apply

A second `plan` or `apply` with unchanged config and unchanged on-disk state SHALL
report no changes and SHALL NOT touch any file or the secret backend, including
for secret-backed values.

#### Scenario: Second apply is a no-op
- **WHEN** apply runs twice with no config change
- **THEN** the second run prints "No changes" and modifies nothing

#### Scenario: Secret-backed value stays idempotent
- **WHEN** a config value is a secret reference that was already applied
- **THEN** the next plan reports it as a noop (no spurious update) without
  re-resolving the secret

### Requirement: Drift detection

`homonto status` SHALL report managed keys whose on-disk value diverges from the
last-applied snapshot recorded in state.

#### Scenario: Out-of-band change surfaces
- **WHEN** a managed key is changed on disk outside homonto after an apply
- **THEN** `status` lists that key as drifted

#### Scenario: No drift after clean apply
- **WHEN** no on-disk managed value has changed since the last apply
- **THEN** `status` reports no drift

