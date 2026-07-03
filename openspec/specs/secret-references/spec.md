# secret-references Specification

## Purpose
TBD - created by archiving change homonto-v1-core. Update Purpose after archive.
## Requirements
### Requirement: Secrets are referenced, never stored

Secret values SHALL be expressed in `homonto.toml` only as references
(`${pass:PATH}` resolved via `pass`, or `${ENV}` resolved from the environment).
Plaintext secret values SHALL never be required in the repo.

#### Scenario: Pass reference resolves at apply
- **WHEN** a value is `${pass:ai/brave}` and apply is confirmed
- **THEN** the resolver invokes the `pass` backend for `ai/brave` and substitutes
  the returned value only into the file being written

#### Scenario: Env reference resolves from environment
- **WHEN** a value is `${BRAVE_API_KEY}` and the variable is set
- **THEN** the resolver substitutes the environment value

#### Scenario: Missing reference errors by name
- **WHEN** a referenced env var is unset or a `pass` path is absent
- **THEN** resolution fails with an error naming the missing reference

### Requirement: Plan output never contains a resolved secret

Plan and log output SHALL display secret-bearing values only as their unresolved
tokens, never as resolved plaintext — including when a secret-backed key is
created, updated, or reported as drifted.

#### Scenario: Create shows the token
- **WHEN** a plan creates a key whose value is `${pass:ai/brave}`
- **THEN** the output contains `${pass:ai/brave}` and never the resolved value

#### Scenario: Drift of a secret value is redacted
- **WHEN** a secret-backed key has drifted on disk and the plan shows an update
- **THEN** the change's old value is redacted (e.g. `«secret»`) and the resolved
  on-disk secret never appears in the output

### Requirement: State stores unresolved token plus a non-secret hash

For each managed key, state SHALL store the unresolved desired value and a
non-secret hash (sha256) of the resolved value written to disk. `state.json` SHALL
NOT contain any plaintext secret and SHALL remain safe to share.

#### Scenario: State records desired token and applied hash
- **WHEN** a secret-backed change is applied
- **THEN** state stores the `${pass:…}` token and `sha256(resolved value)`, not the
  resolved value

#### Scenario: Idempotency decision uses token match plus hash
- **WHEN** planning a secret-backed key that is present in state
- **THEN** it is a noop only if the desired token matches state and
  `sha256(on-disk value)` matches the stored hash; otherwise it is an update

#### Scenario: State file has no plaintext secret
- **WHEN** `state.json` is read after any apply
- **THEN** it contains no resolved secret value

