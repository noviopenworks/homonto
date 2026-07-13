# cli-commands (delta)

## ADDED Requirements

### Requirement: no clean conclusion after incomplete coverage

`homonto plan` and `status` SHALL NOT print a clean conclusion ("Everything up to
date" / "No drift") or exit zero when any adapter warning was emitted during the
run — a warning means a tool was skipped or only partially observed, so coverage
was incomplete. In that case the command SHALL exit non-zero and report that
coverage was incomplete (the warnings are still printed), matching the guard
`apply` already applies to a skipped adapter.

#### Scenario: plan does not claim up-to-date after a warning

- **GIVEN** a run where an adapter emitted a warning and produced no projected changes
- **WHEN** `homonto plan` runs
- **THEN** it does not print "Everything up to date", it reports incomplete coverage, and it exits non-zero
