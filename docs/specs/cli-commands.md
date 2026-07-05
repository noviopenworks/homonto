# cli-commands Specification

## Purpose
Defines the user-facing command surface and each command's safety behavior,
including initialization, import, plan/apply/status, health checks, and version
reporting.
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

`homonto import` SHALL read Claude Code global MCP servers (`~/.claude.json`
`mcpServers`) into a starter `homonto.toml`, reading each MCP entry in the real
schema — `command` as a string plus an `args` array — while tolerating the legacy
all-in-`command` array form, and preserving the full argv into the generated
config. OpenCode import, Claude settings/plugins/skills import, and non-stdio
servers are not implemented and MUST NOT be claimed. Non-stdio servers
(url/http) SHALL be skipped with a warning, never imported as empty commands.
Env values that look like literal secrets SHALL be replaced with a `${pass:…}`
reference and reported as a warning; command and args values are currently
preserved as-is, so users SHOULD review generated config before sharing it. A
tool file that exists but cannot be read or parsed SHALL produce a warning,
never silent omission. Import SHALL refuse to overwrite an existing config
unless `--force` is given.

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

#### Scenario: Command arguments are not redacted

- **WHEN** an imported MCP command or args entry contains a literal secret
- **THEN** import preserves it verbatim in the generated config; this is a known
  limitation and the user must review the file before sharing it

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
`content/skills/`. For every owned skill it SHALL verify BOTH tool symlinks —
`~/.claude/skills/<name>` and `~/.config/opencode/skills/<name>` — reporting the
link state per tool. All findings are reported as `ok`/`warn` lines.

#### Scenario: Missing owned skill is flagged
- **WHEN** a skill listed in `[skills] own` has no directory under
  `content/skills/`
- **THEN** `doctor` reports a warning naming that skill

#### Scenario: Missing OpenCode link is flagged
- **GIVEN** an owned skill whose content exists and whose Claude link is
  correct but whose OpenCode link is missing
- **WHEN** `doctor` runs
- **THEN** it reports the Claude link as `ok` and warns that the skill is not
  linked for `opencode`

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
