## ADDED Requirements

### Requirement: Command surface

`homonto` SHALL expose `version`, `init`, `import`, `plan`, `apply`, `status`, and
`doctor`, with a persistent `--config` flag (default `homonto.toml`). Config
changes happen by editing `homonto.toml`; there SHALL be no imperative
`add`/`remove` mutators in v1.

#### Scenario: Version prints the build version
- **WHEN** the user runs `homonto version`
- **THEN** it prints `homonto <version>`

### Requirement: init scaffolds without overwriting

`homonto init [dir]` SHALL scaffold a starter repo (`homonto.toml`, `.gitignore`,
`.env.example`, `content/skills/`) and SHALL never overwrite an existing file.

#### Scenario: Existing files are preserved
- **WHEN** `homonto.toml` already exists in the target dir
- **THEN** `init` leaves it unchanged and only creates the missing files

### Requirement: import bootstraps with secret redaction

`homonto import` SHALL read the current Claude/OpenCode setup into a starter
`homonto.toml`, replacing any value that looks like a literal secret with a
`${pass:…}` reference and reporting a warning. It SHALL refuse to overwrite an
existing config unless `--force` is given. No literal secret SHALL be written to
the generated config.

#### Scenario: Literal secret is redacted
- **WHEN** an imported env value looks like a secret (e.g. `sk-…`, or a
  `*_KEY`/`*_TOKEN` key with a non-reference value)
- **THEN** it is replaced with a `${pass:…}` reference, a warning is emitted, and
  the literal secret never appears in the output

#### Scenario: Overwrite guarded
- **WHEN** a config already exists and `--force` is not given
- **THEN** import refuses and reports, leaving the existing config unchanged

### Requirement: doctor health checks

`homonto doctor` SHALL check that `pass` is on `PATH`, that each target tool's
config location is present, and that each owned skill exists under
`content/skills/`, reporting `ok`/`warn` lines.

#### Scenario: Missing owned skill is flagged
- **WHEN** a skill listed in `[skills] own` has no directory under
  `content/skills/`
- **THEN** `doctor` reports a warning naming that skill
