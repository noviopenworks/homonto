# remote-source-trust (delta)

## ADDED Requirements

### Requirement: remote locators never leak embedded credentials

`homonto` SHALL NOT write a remote locator's embedded credentials (URL userinfo
such as `user:pass@`, or a secret query token) verbatim into `remote.lock.json`
or into any error message or log line. `homonto` SHALL either reject a locator
with embedded credentials at load, or store and report only a redacted canonical
form (credentials removed), so the lockfile and diagnostics never carry the
secret.

#### Scenario: a credential in a locator does not reach the lockfile or errors

- **GIVEN** a remote source `https://user:s3cret@host/repo` (or a token query param)
- **WHEN** `homonto` processes it (including on a verification error)
- **THEN** neither `remote.lock.json` nor any emitted error/log contains `s3cret`; the recorded/reported locator is redacted
