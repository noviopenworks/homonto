# Delta Spec: cli-commands (doctor-opencode-link)

## MODIFIED Requirements

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
