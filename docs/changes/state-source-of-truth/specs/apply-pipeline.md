# Delta Spec: apply-pipeline (state-source-of-truth)

## ADDED Requirements

### Requirement: State adoption on apply

`homonto apply` SHALL adopt a declared, non-secret managed key that is present
on disk, equal to its desired value, and absent from state by recording it in
state (an `Applied` hash equal to the on-disk value) **without writing the
tool file and without printing a plan diff line**. Apply SHALL perform this
adoption even when adopting pre-existing keys is the only pending work, in
which case it SHALL NOT prompt for confirmation (only `state.json` is touched)
and SHALL report how many resources were reconciled. Secret-bearing keys SHALL
never be adopted.

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

#### Scenario: Adopted key becomes pruneable

- **GIVEN** a key that was adopted into state on a prior apply
- **WHEN** it is removed from `homonto.toml` and the user runs plan and apply
- **THEN** the plan shows a delete for that key and apply removes it from the
  tool file and from state

## MODIFIED Requirements

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
