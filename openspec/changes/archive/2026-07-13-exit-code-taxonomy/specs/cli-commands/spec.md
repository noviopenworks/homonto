# cli-commands (delta)

## ADDED Requirements

### Requirement: opt-in exit-code taxonomy for plan and status

`homonto plan` and `status` SHALL accept an opt-in `--exit-code` flag. WITHOUT the
flag, exit behavior is unchanged (0 on success, 1 on error). WITH `--exit-code`,
the command SHALL exit with a documented code so automation can branch on state:
`0` = no pending changes and no drift; `2` = pending changes; `3` = drift detected
(status). An error SHALL still exit `1`. Adding the flag SHALL NOT change the
command's printed output or its default (flagless) exit behavior.

#### Scenario: plan --exit-code signals pending changes

- **GIVEN** a config with pending changes and `--exit-code`
- **WHEN** `homonto plan --exit-code` runs
- **THEN** it exits with code 2 (pending), while `homonto plan` without the flag exits 0

#### Scenario: default exit behavior is unchanged

- **WHEN** `homonto plan` or `status` runs WITHOUT `--exit-code`
- **THEN** it exits 0 on success regardless of pending changes or drift (unchanged)
