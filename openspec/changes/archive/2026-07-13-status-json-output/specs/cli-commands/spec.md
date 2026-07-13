# cli-commands (delta)

## ADDED Requirements

### Requirement: status supports machine-readable JSON output

`homonto status` SHALL accept `--output text|json` (default `text`). With
`--output json` it SHALL emit a single JSON object carrying the drift lines, the
pending-change count, and the warnings, so automation can consume status without
scraping human-formatted text. An unrecognized `--output` value SHALL be rejected.
The default `text` output is unchanged.

#### Scenario: status --output json emits parseable JSON

- **WHEN** the user runs `homonto status --output json`
- **THEN** the output is a single JSON object with `drift`, `pending`, and `warnings` fields
