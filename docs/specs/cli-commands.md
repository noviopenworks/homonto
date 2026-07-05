# cli-commands Specification

## Purpose
TBD - created by archiving change homonto-v1-core. Update Purpose after archive.
## Requirements

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

`homonto import` SHALL read the current Claude Code setup (`~/.claude.json`
MCP servers) into a starter `homonto.toml`, reading each MCP entry in the
real schema — `command` as a string plus an `args` array — while tolerating
the legacy all-in-`command` array form, and preserving the full argv into
the generated config; OpenCode import is not implemented and MUST NOT be
claimed. Non-stdio servers (url/http) SHALL be skipped with a warning,
never imported as empty commands. Any value that looks like a literal secret SHALL be replaced with a
`${pass:…}` reference and reported as a warning; the redaction heuristics
SHALL cover at least the value prefixes `sk-`, `ghp_`, `github_pat_`, `xox`,
`glpat-`, `npm_`, `AIza`, `Bearer ` and the key patterns `*_KEY`, `*_TOKEN`,
`*_SECRET`, `*_PASSWORD`, `*_CREDENTIALS`, `DATABASE_URL`. A tool file that
exists but cannot be read or parsed SHALL produce a warning, never silent
omission. It SHALL refuse to overwrite an existing config unless `--force`
is given. No literal secret SHALL be written to the generated config.

#### Scenario: Real schema imported with args preserved

- **GIVEN** a `~/.claude.json` MCP entry with `"command": "npx"` and
  `"args": ["-y", "some-server"]`
- **WHEN** the user runs `homonto import`
- **THEN** the generated config's command is `["npx", "-y", "some-server"]`
  — no argument is dropped

#### Scenario: Literal secret is redacted

- **WHEN** an imported env value looks like a secret (e.g. `sk-…`, `glpat-…`,
  or a `*_KEY`/`*_TOKEN`/`*_SECRET`/`*_PASSWORD` key with a non-reference
  value)
- **THEN** it is replaced with a `${pass:…}` reference, a warning is emitted, and
  the literal secret never appears in the output

#### Scenario: Unreadable tool file warns

- **GIVEN** a tool config file that exists but cannot be read or parsed
- **WHEN** the user runs `homonto import`
- **THEN** a warning naming the file is emitted instead of silently
  skipping it

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

### Requirement: Version reporting

`homonto --version` SHALL print the build version from a package-level
`var` (not a constant) so release builds can stamp it at link time via
`-ldflags "-X …"`, with a recognizable dev default otherwise.

#### Scenario: Stamped version printed

- **GIVEN** a binary built with `-ldflags "-X <module>/internal/cli.Version=1.2.3"`
- **WHEN** the user runs `homonto --version`
- **THEN** the output contains `1.2.3`

#### Scenario: Dev build identifies itself

- **GIVEN** a binary built without ldflags stamping
- **WHEN** the user runs `homonto --version`
- **THEN** the output contains the dev default version
