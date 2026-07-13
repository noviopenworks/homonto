# cli-commands (delta)

## ADDED Requirements

### Requirement: plan supports machine-readable JSON output

`homonto plan` SHALL accept `--output text|json` (default `text`). With `--output
json` it SHALL emit a single JSON object describing the pending visible changes as
`{action, key}` per tool, the pending remote repins, and any warnings. It SHALL
NOT include the Old/New change values in JSON (they can carry unresolved secret
tokens), so the machine output never leaks a secret reference's context. An
unrecognized `--output` value SHALL be rejected; the default `text` output is
unchanged.

#### Scenario: plan --output json emits parseable JSON

- **WHEN** the user runs `homonto plan --output json`
- **THEN** the output is a single JSON object with `changes`, `repins`, and `warnings` fields, and no `old`/`new` value strings
