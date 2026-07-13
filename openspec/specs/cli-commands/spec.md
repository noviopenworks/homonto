# cli-commands Specification

## Purpose
Defines the user-facing command surface and each command's safety behavior,
including initialization, import, plan/apply/status, health checks, and version
reporting. There is no imperative agent command group; `[agents.<name>]` is a
deprecated alias folded into a subagent at config load.
## Requirements
### Requirement: Command surface

`homonto` SHALL expose the top-level commands `version`, `init`, `import`,
`plan`, `apply`, `status`, and `doctor`, with a persistent `--config` flag
(default `homonto.toml`). MCP servers, settings, plugins, marketplaces, TUI
settings, skills, commands, subagents, and frameworks are reconciled
declaratively through the `plan`/`apply` model by editing `homonto.toml`. The
deprecated `[agents.<name>]` table is also handled declaratively: it is folded
into an equivalent copy-mode subagent at config load and projected by `apply`
like any other subagent. There is no imperative `agents` command group.

#### Scenario: Version prints the build version
- **WHEN** the user runs `homonto version`
- **THEN** it prints `homonto <version>`

#### Scenario: Only declarative commands are registered
- **WHEN** the user runs `homonto --help`
- **THEN** it lists exactly `version`, `init`, `import`, `plan`, `apply`,
  `status`, and `doctor`
- **AND** no `agents` command group is present

### Requirement: init scaffolds without overwriting

`homonto init [dir]` SHALL scaffold a starter repo (`homonto.toml`, `.gitignore`,
`.env.example`, `homonto/skills/`) and SHALL never overwrite an existing file.

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
config location is present, and that each local-source owned resource — skill,
command, and subagent — exists under its `homonto/` provider root
(`homonto/skills/<name>`, `homonto/commands/<name>.md`,
`homonto/subagents/<name>.md`). For every owned skill, command, and subagent it
SHALL verify BOTH tool links at the location selected by that resource's `scope`,
for Claude Code and OpenCode alike:

- skills: `~/.claude/skills/<name>` and `~/.config/opencode/skills/<name>`
  (`user`), or `<project>/.claude/skills/<name>` and
  `<project>/.opencode/skills/<name>` (`project`);
- commands: `~/.claude/commands/<name>.md` and
  `~/.config/opencode/command/<name>.md` (`user`), or the `<project>/.claude/…`
  and `<project>/.opencode/…` equivalents (`project`);
- subagents: `~/.claude/agents/<name>.md` and
  `~/.config/opencode/agent/<name>.md` (`user`), or the `<project>/.claude/…`
  and `<project>/.opencode/…` equivalents (`project`).

Builtin resources (`source = "builtin:<name>"`) resolve from the versioned
materialized catalog at `.homonto/catalog/skills/<name>/`,
`.homonto/catalog/commands/<name>.md`, and
`.homonto/catalog/subagents/<name>.md`; `doctor` SHALL flag a builtin resource
whose materialized target is missing or whose recorded catalog version differs
from the embedded catalog version. All findings are reported as `ok`/`warn`
lines.

#### Scenario: Missing owned skill is flagged
- **WHEN** a declared local-source skill resource has no directory under
  `homonto/skills/`
- **THEN** `doctor` reports a warning naming that skill

#### Scenario: Missing OpenCode link is flagged
- **GIVEN** an owned skill whose content exists and whose Claude link is
  correct but whose OpenCode link is missing
- **WHEN** `doctor` runs
- **THEN** it reports the Claude link as `ok` and warns that the skill is not
  linked for `opencode`

#### Scenario: Project scope is checked at the project location
- **GIVEN** a config with `[skills.<name>] scope = "project"` whose skills are
  applied
- **WHEN** `doctor` runs
- **THEN** it reports the skill links `ok` by checking `<project>/.claude/skills/<name>` and
  `<project>/.opencode/skills/<name>`, not the home locations

#### Scenario: Command and subagent links are verified for both tools
- **GIVEN** a config declaring an owned command and an owned subagent, each
  applied for both Claude Code and OpenCode
- **WHEN** `doctor` runs
- **THEN** it reports the command and subagent links `ok` for both tools, and
  flags any missing or incorrect link for either tool with a `warn` line

#### Scenario: Builtin resource materialization is verified
- **GIVEN** a builtin skill, command, or subagent whose materialized target under
  `.homonto/catalog/` is missing
- **WHEN** `doctor` runs
- **THEN** it reports a warning naming the resource and its missing materialized
  target

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

### Requirement: positional-free commands reject stray arguments

`homonto plan`, `apply`, `status`, `doctor`, and `import` SHALL reject unexpected
positional arguments (`cobra.NoArgs`) with a non-zero exit and a clear error,
rather than silently ignoring them, so a user who runs e.g. `homonto apply
production.toml` is told the file was not consumed (config is selected only via
`--config`). `homonto init` keeps its single optional positional (target dir).

#### Scenario: a stray positional is rejected

- **WHEN** the user runs `homonto apply production.toml` (a stray positional)
- **THEN** the command exits non-zero with an "unknown command / unexpected argument" error and does not run apply against the default config

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

### Requirement: cache gc reclaims unreferenced remote cache entries

`homonto cache gc [--dry-run]` SHALL reclaim content-addressed remote cache entries
that no entry in the remote lockfile references, and SHALL report the digests it
removed. With `--dry-run` it SHALL report what it would remove without deleting
anything. The command SHALL reject stray positional arguments.

#### Scenario: dry-run reports without deleting

- **WHEN** the user runs `homonto cache gc --dry-run`
- **THEN** it reports the unreferenced entries it would reclaim and deletes nothing

### Requirement: import backs up an existing config before overwriting

`homonto import --force` SHALL, before overwriting an existing config file, copy
the existing file to `<config>.bak`, and SHALL write the new config atomically, so
a forced import over a valid config is recoverable and never leaves a partially
written file.

#### Scenario: forced import over an existing config preserves a backup

- **GIVEN** an existing `homonto.toml` with valid content
- **WHEN** the user runs `homonto import --force`
- **THEN** the previous content is preserved at `homonto.toml.bak` and the new config is written atomically
