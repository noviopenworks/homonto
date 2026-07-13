# remote-source-trust (delta)

## ADDED Requirements

### Requirement: remotes verify into staging before any active mutation

`homonto` SHALL fetch and verify every declared remote source into a staging area
before it prunes, mutates, or unlinks any active remote content or the remote
lockfile. If any remote fails to fetch or verify, `homonto` SHALL leave all active
remote content and the lockfile unchanged (all-or-nothing across remotes).

#### Scenario: a later remote failure leaves earlier content and the lock intact

- **GIVEN** two declared remotes where the second fails verification
- **WHEN** `homonto apply` materializes remotes
- **THEN** the first remote's active content and the lockfile are unchanged, and apply reports the failure without a partial mutation

### Requirement: git fetch is bounded before checkout

`homonto` SHALL run a remote git fetch under a deadline (a bounded context, not
`context.Background()`) and SHALL enforce the size and file-count limits at or
before checkout, so a malicious pinned repository cannot exhaust time or disk
before validation.

#### Scenario: an oversized remote is rejected before it exhausts disk

- **GIVEN** a pinned remote exceeding the configured size/file caps
- **WHEN** `homonto` fetches it
- **THEN** the fetch is aborted under the deadline/limits before full checkout, with a clear error

### Requirement: doctor verifies materialized remote digests and revocation deactivates

`homonto doctor` SHALL verify each materialized remote content's digest against the
lockfile and report a mismatch as a finding. Revoked-but-still-declared content
SHALL be deactivated (not left linked) after a failed or revoked apply.

#### Scenario: a materialized-digest mismatch is a doctor finding

- **GIVEN** materialized remote content whose bytes no longer match its lockfile digest
- **WHEN** `homonto doctor` runs
- **THEN** it reports the digest mismatch as a finding
