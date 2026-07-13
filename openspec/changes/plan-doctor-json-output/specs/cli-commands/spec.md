# cli-commands (delta)

## ADDED Requirements

### Requirement: doctor supports machine-readable JSON output

`homonto doctor` SHALL accept `--output text|json` (default `text`). With
`--output json` it SHALL emit a single JSON object with a `findings` array (the
same lines the text output prints), so automation can consume health findings
without scraping text. An unrecognized `--output` value SHALL be rejected; the
default `text` output is unchanged.

#### Scenario: doctor --output json emits parseable findings

- **WHEN** the user runs `homonto doctor --output json`
- **THEN** the output is a single JSON object with a `findings` array
