## ADDED Requirements

### Requirement: Every declared subagent resolves an explicit per-tool model
The loader SHALL require, for every declared subagent and every target tool
enabled for that subagent, a `[subagents.<name>.<tool>]` block whose `model` is
non-empty. When a declared subagent lacks a model for an enabled tool, config
load SHALL fail with an error that names both the subagent and the tool
(`subagents.<name>.<tool> model is required`). A subagent SHALL NOT inherit a
model from any shared default, tier, or role.

#### Scenario: All agents modeled — load succeeds
- **WHEN** the config declares subagents `onto`, `onto-explorer`,
  `onto-reviewer`, `onto-implementer`, `onto-skeptic`, `to-explorer`,
  `to-implementer`, `to-reviewer`, `to-skeptic`, and each has a
  `[subagents.<name>.claude]` and `[subagents.<name>.opencode]` block with a
  non-empty `model`
- **THEN** `homonto doctor` exits 0 and every agent renders with its declared
  model

#### Scenario: Missing agent model — load fails naming the offender
- **WHEN** the config declares subagent `onto-skeptic` but no
  `[subagents.onto-skeptic.opencode]` block (or its `model` is empty), and
  opencode is an enabled tool for that subagent
- **THEN** config load fails with `subagents.onto-skeptic.opencode model is
  required` before any agent file is rendered

#### Scenario: Framework declares existence, config declares model
- **WHEN** the onto framework's `framework.toml` declares subagent
  `onto-reviewer` and the installer's config provides
  `[subagents.onto-reviewer.claude] model = "opus"`
- **THEN** the agent is rendered with model `opus`; the framework declaration
  alone contributes no model and the config declaration alone contributes no
  existence

### Requirement: No tier or role indirection
The config SHALL NOT define `[models.<tool>.<tier>]` blocks for any tier name
(`architectural`, `coding`, `review`, `trivial`, or otherwise). Subagent
frontmatter SHALL NOT carry a `role:` field. Model selection SHALL depend only
on the explicit `[subagents.<name>.<tool>]` block (plus an explicit
`[settings.<tool>]` override for the main session). A config that contains a
`[models.<tool>.<tier>]` block SHALL fail validation naming the unknown key.

#### Scenario: Legacy tier block rejected
- **WHEN** the config contains `[models.claude.architectural]`
- **THEN** config load fails with an error naming `models.claude.architectural`
  as an unknown key, and no agent is rendered

#### Scenario: role frontmatter ignored or rejected
- **WHEN** a catalog subagent `.md` file contains `homonto: role: coding`
- **THEN** the renderer treats `role` as unknown frontmatter (ignored or
  rejected per the frontmatter policy) and does NOT derive a model from it

### Requirement: The main session model is operator-controlled, not homonto-managed
Homonto SHALL NOT project a default model into the main Claude or OpenCode
session settings. The main session model SHALL be whatever the target tool
itself defaults to. The route-derived default-model projection (Claude
`routeModelSetting`, OpenCode `routeSettings` model/small_model derivation)
SHALL be removed.

#### Scenario: No main model written when settings absent
- **WHEN** the config declares no `[settings.claude].model` and no
  `[settings.opencode].model`
- **THEN** homonto writes no `model` key into Claude's settings or OpenCode's
  settings; each tool uses its own default model

#### Scenario: Explicit main-model override still projected
- **WHEN** the config declares `[settings.opencode].model = "anthropic/claude-opus-4-8"`
- **THEN** homonto projects that value into OpenCode's settings as before; the
  explicit-settings path is unchanged by this change

### Requirement: Per-tool model spec validation is preserved
A `[subagents.<name>.<tool>]` block MAY set `effort` (Claude only, one of
`low|medium|high|xhigh|max`) and `variant` (Claude model alias, or OpenCode
variant), and SHALL NOT set `effort` for OpenCode. The loader SHALL reject any
effort/variant value the target tool cannot express, naming the offending
block. `model` is required; `effort` and `variant` are optional.

#### Scenario: Invalid Claude effort rejected
- **WHEN** `[subagents.onto-implementer.claude]` sets `effort = "ludicrous"`
- **THEN** config load fails naming `subagents.onto-implementer.claude` and the
  invalid effort value

#### Scenario: Effort on OpenCode rejected
- **WHEN** `[subagents.onto-explorer.opencode]` sets `effort = "low"`
- **THEN** config load fails, because OpenCode accepts no effort setting
