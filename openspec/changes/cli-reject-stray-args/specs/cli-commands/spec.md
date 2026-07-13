# cli-commands (delta)

## ADDED Requirements

### Requirement: positional-free commands reject stray arguments

`homonto plan`, `apply`, `status`, `doctor`, and `import` SHALL reject unexpected
positional arguments (`cobra.NoArgs`) with a non-zero exit and a clear error,
rather than silently ignoring them, so a user who runs e.g. `homonto apply
production.toml` is told the file was not consumed (config is selected only via
`--config`). `homonto init` keeps its single optional positional (target dir).

#### Scenario: a stray positional is rejected

- **WHEN** the user runs `homonto apply production.toml` (a stray positional)
- **THEN** the command exits non-zero with an "unknown command / unexpected argument" error and does not run apply against the default config
