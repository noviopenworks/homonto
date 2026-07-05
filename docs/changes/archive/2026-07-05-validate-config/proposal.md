Preset: fix

# Proposal: validate-config

## Why

`config.Load` validates skill names and JSON key names, but three classes of
invalid `homonto.toml` input are accepted and then **silently ignored** at
projection time instead of failing fast (NEXT_AGENT gap #3). A typo produces
no error and no effect — the worst outcome for a declarative tool.

## The bug (reproduction / expected vs actual)

1. **Unknown target name.** An MCP with `targets = ["claud"]` (typo) matches
   neither adapter (`contains(targets, "claude"/"opencode")` is false), so it
   is projected to no tool. *Actual:* silent no-op. *Expected:* config load
   fails naming the unknown target and the valid set `{claude, opencode}`.
2. **Empty command.** An MCP with no `command` (or `command = []`) is skipped
   by both adapters' `desired()` (`len(m.Command) == 0` → continue). *Actual:*
   silent no-op. *Expected:* config load fails naming the MCP that cannot
   project.
3. **Reserved settings namespace.** A `settings.claude` key `enabledPlugins`,
   or a `settings.opencode` key `mcp` or `plugin`, collides with homonto's own
   managed structures in the same tool file. *Actual:* accepted, then fights
   homonto's writes. *Expected:* config load fails naming the reserved key.

## Fix scope

Add validation in `internal/config/config.go` `Load` (fail fast, offender
named), covering: MCP target names ∈ {claude, opencode}; non-empty MCP
command; reserved settings keys (`settings.claude.enabledPlugins`,
`settings.opencode.mcp`, `settings.opencode.plugin`). Tests for each. A small
`config-model` delta spec captures the new validation requirements.

## Capability Impact

- **Modified**: `config-model` — adds input-validation requirements (delta).
- Untouched: apply-pipeline, tool-adapters, secret-references, cli-commands,
  onto-workflow.

## Grounding

`internal/config/config.go:50-108` (`Load` + `validateKey`, no target/command/
reserved checks); `MCP.TargetsOrAll` :20-25; adapters skip empty command at
`claude.go` `desired()` `len(m.Command)==0` and `opencode.go` `desiredMCPs`.
Managed settings collisions: claude writes `enabledPlugins` into
`settings.json`; opencode writes `mcp`/`plugin` into `opencode.jsonc`.

## Impact

- Files: `internal/config/config.go`, `internal/config/*_test.go`, delta spec
  `specs/config-model.md`. ≤5 non-test files — no upgrade trigger.
- Risk: a validation that is too strict could reject a currently-valid config.
  Mitigated by targeting only the three documented invalid classes and adding
  a test that a valid multi-target / multi-setting config still loads.
