# apply-pipeline Specification

## Purpose
Defines the plan/confirm/apply pipeline, including dry-run planning, confirmed
secret resolution, atomic writes, per-adapter state persistence, idempotency,
and the current status/drift behavior.
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

### Requirement: State adoption on apply

`homonto apply` SHALL adopt a declared, non-secret managed key that is present
on disk and equal to its desired value but whose recorded applied-value hash is
absent or stale, by recording (or refreshing) its state entry with an `Applied`
hash equal to the on-disk value — without writing the tool file and without
printing a plan diff line. Apply SHALL perform this adoption even when adopting
such keys is the only pending work, in which case it SHALL NOT prompt for
confirmation (only `state.json` is touched) and SHALL report how many resources
were reconciled. Secret-bearing keys SHALL never be adopted.

#### Scenario: Pre-existing matching key adopted on apply

- **GIVEN** a declared non-secret key whose on-disk value already equals
  desired and which has no record in state
- **WHEN** the user runs `homonto apply`
- **THEN** the plan shows no diff line for that key, the tool file is left
  byte-unchanged, and after apply `state.json` records the key with an applied
  hash of its on-disk value

#### Scenario: Adoption-only apply runs without a prompt

- **GIVEN** a config whose declared keys all already match disk, one of which
  is not yet recorded in state
- **WHEN** the user runs `homonto apply`
- **THEN** apply reconciles state without asking for `[y/N]` confirmation,
  reports the number of pre-existing resources reconciled, and a second apply
  reports `No changes`

#### Scenario: Stale applied hash is refreshed, clearing phantom drift

- **GIVEN** a recorded key whose on-disk value was changed out of band to a
  value that now equals the desired value (so its `Applied` hash is stale)
- **WHEN** the user runs `homonto apply`
- **THEN** the key's state `Applied` hash is refreshed to the on-disk value
  with no tool-file write, and a subsequent `status` no longer reports it as
  drifted

#### Scenario: Adopted key becomes pruneable

- **GIVEN** a key that was adopted into state on a prior apply
- **WHEN** it is removed from `homonto.toml` and the user runs plan and apply
- **THEN** the plan shows a delete for that key and apply removes it from the
  tool file and from state

### Requirement: Drift detection

`homonto status` SHALL report a state-recorded managed key as drifted only when
its current on-disk value differs from the last-applied value recorded in state
(the `Applied` hash), or when the key is missing from disk entirely. Edits to
`homonto.toml` that have not yet been applied SHALL NOT be reported as drift;
`status` SHALL instead report them separately as a count of config changes
awaiting apply.

#### Scenario: Out-of-band change surfaces

- **WHEN** a managed key is changed on disk outside homonto after an apply
- **THEN** `status` lists that key as drifted

#### Scenario: No drift after clean apply

- **WHEN** no on-disk managed value has changed since the last apply
- **THEN** `status` reports no drift

#### Scenario: Config edit is pending, not drift

- **GIVEN** a key recorded in state and a later `homonto.toml` edit changing
  that key's desired value, with the on-disk value unchanged since the last
  apply
- **WHEN** `status` runs before `apply`
- **THEN** the key is not reported as drifted, and `status` reports one config
  change awaiting apply

#### Scenario: Deleted managed key reported missing

- **GIVEN** a key recorded in state whose value was removed from disk out of
  band
- **WHEN** `status` runs
- **THEN** `status` reports that key as missing

### Requirement: Single secret resolution per run

The secret resolver SHALL memoize by token within a run: each distinct
reference token SHALL trigger at most one backend invocation per plan or
apply run, however many keys share it.

#### Scenario: Shared token resolved once

- **GIVEN** two config values referencing the same `${pass:ai/brave}` token
- **WHEN** apply resolves secrets for confirmed changes
- **THEN** the `pass` backend is invoked exactly once for that token and
  both keys receive the resolved value

### Requirement: Remote resolution routes through the trust pipeline

When the apply pipeline resolves a resource whose source is `remote:`, it SHALL
route resolution through the remote trust pipeline (cache lookup → verified
fetch → validate → pin-match → revocation) and materialize only from the
content-addressed cache. `builtin:` and `local:` resolution SHALL be unchanged.
A remote resolution failure SHALL abort the apply before any target mutation,
consistent with the atomic-writes / state-last guarantee.

#### Scenario: Remote resource projects like a managed resource

- **GIVEN** a pinned, cacheable `remote:` subagent/skill/command
- **WHEN** plan then apply runs
- **THEN** it materializes into each target tool exactly like a builtin/local resource, and status/doctor track it

#### Scenario: Remote resolution failure aborts apply cleanly

- **GIVEN** a `remote:` resource whose content fails verification
- **WHEN** apply runs
- **THEN** the apply aborts before any target file is written and existing state is unchanged

#### Scenario: Idempotent remote apply

- **GIVEN** an already-applied pinned remote resource
- **WHEN** apply runs again
- **THEN** it is a no-op (cache hit, no network, no target rewrite)
