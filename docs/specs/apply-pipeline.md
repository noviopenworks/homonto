# apply-pipeline Specification

## Purpose
TBD - created by archiving change homonto-v1-core. Update Purpose after archive.
## Requirements

### Requirement: Plan is a pure dry run

`homonto plan` SHALL compute and print the diff between desired and current state
without writing any file, resolving any secret, or contacting the secret backend.
When a managed key is present on disk but has no record in state (unknown
provenance), the plan SHALL redact the on-disk old value instead of printing it —
unknown provenance is treated as secret.

#### Scenario: Plan writes nothing

- **WHEN** the user runs `homonto plan`
- **THEN** a terraform-style diff is printed and no tool file, symlink, or state
  file is created or modified

#### Scenario: Plan shows creates and updates, hides noops

- **WHEN** the plan contains create, update, and noop changes
- **THEN** the output shows `+` for creates and `~` for updates and omits noops

#### Scenario: Unknown-provenance old value redacted

- **GIVEN** a key that exists on disk with a value differing from desired,
  and no record of that key in state
- **WHEN** the plan shows the update
- **THEN** the old value is displayed redacted (e.g. `«secret»`) and the
  raw on-disk value never appears in the output

### Requirement: Apply is confirmation-gated and two-phase

`homonto apply` SHALL print the plan, require confirmation (`[y/N]`, skippable
with `--yes`), and then apply in two phases: resolve **all** secrets for confirmed
changes first, and only if every resolution succeeds proceed to write. State SHALL
be persisted per adapter: after each adapter's writes succeed its records are
saved, so a failure in a later adapter keeps every earlier adapter's applied
records, and apply exits non-zero naming the failing adapter.

#### Scenario: Confirmation declined

- **WHEN** the user answers anything other than `y`
- **THEN** apply aborts and no file is written

#### Scenario: Missing secret aborts before any write

- **WHEN** a confirmed change references a secret that cannot be resolved
- **THEN** apply aborts before writing any file, names the missing reference, and
  leaves every tool file and the state file unchanged

#### Scenario: Partial apply keeps earlier adapters' records

- **GIVEN** two adapters where the first applies cleanly and the second fails
- **WHEN** apply runs
- **THEN** the first adapter's changes are recorded in state, apply exits
  non-zero naming the failing adapter, and the next plan does not re-show
  the first adapter's already-applied changes

### Requirement: Atomic writes

Every file write SHALL go through a temp file followed by rename, so an
interrupted apply never leaves a half-written file; state SHALL be written
only after the tool files it records — per adapter, each adapter's state save
follows that adapter's successful writes and never precedes them.

#### Scenario: Crash-safety ordering

- **WHEN** apply writes an adapter's tool files and then its state records
- **THEN** each tool file is individually valid at all times and that
  adapter's state is written only after its tool files succeed

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

### Requirement: Single secret resolution per run

The secret resolver SHALL memoize by token within a run: each distinct
reference token SHALL trigger at most one backend invocation per plan or
apply run, however many keys share it.

#### Scenario: Shared token resolved once

- **GIVEN** two config values referencing the same `${pass:ai/brave}` token
- **WHEN** apply resolves secrets for confirmed changes
- **THEN** the `pass` backend is invoked exactly once for that token and
  both keys receive the resolved value
