## Why

Homonto's model routing is an indirection: a subagent declares `role: <tier>`,
the tier selects `[models.<tool>.<tier>]`, and `validateModels` requires every
enabled tool to declare **all four** tiers (`architectural`, `coding`,
`review`, `trivial`). That coupling broke the shipped config: code added a
`review` tier and the onto/to subagents reference it, but `homonto.toml` was
never updated — `homonto doctor` now errors with `models.claude.review is
required for enabled target tool "claude"`. The tier system makes any tier
added in code a load-bearing requirement on every installer's config, with no
way for the config to stay correct in isolation.

## What Changes

- **BREAKING** Remove the model tier system entirely: the `role:` frontmatter
  field, the `[models.<tool>.<tier>]` config blocks, `agentfm.TierNames` /
  `Tiers` / `Roles`, and the all-tiers-required `validateModels` check.
- Subagent models move to explicit `[subagents.<name>.<tool>]` blocks in
  `homonto.toml` — one per agent, per tool. The framework declaration
  (`framework.toml [subagents]`) still owns agent **existence**; homonto.toml
  owns agent **models**.
- **BREAKING** Homonto stops managing the main interactive session model.
  Delete `routeModelSetting` (Claude) and the route-derived model half of
  `routeSettings` (OpenCode). The main model becomes the tool's own default,
  overridable via the existing explicit `[settings.claude].model` /
  `[settings.opencode].model` / `.small_model` path.
- Add a must-declare check: every declared subagent must have a
  `[subagents.<name>.<tool>]` model for each enabled tool, else config load
  fails naming the offender.
- Delete the redundant `[subagents.onto-reviewer]` / `[subagents.onto-explorer]`
  source blocks in `homonto.toml` (the onto framework already declares those
  agents — the re-declaration collides today).

## Capabilities

### New Capabilities
- `agent-models`: how homonto assigns models to agents — each declared
  subagent resolves an explicit per-tool model from `[subagents.<name>.<tool>]`;
  there is no tier/role indirection, and homonto does not manage the main
  session model.

### Modified Capabilities
<!-- openspec/specs/ is empty (freshly re-initialized); no existing capabilities to modify. -->

## Impact

- **Config (breaking):** `homonto.toml` loses all `[models.*.*]` blocks and the
  two redundant `[subagents.*]` source blocks; gains 18 `[subagents.<name>.<tool>]`
  blocks (9 agents × 2 tools: onto + to). `homonto doctor` goes green.
- **Code:** `internal/agentfm` (drop Role/TierNames/Tiers/Roles), `internal/config`
  (drop `Models`, rewrite `validateModels` to the must-declare check),
  `internal/adapter/claude` and `internal/adapter/opencode` (drop model
  projection), `internal/scaffold` (template examples), `catalog/subagents/*.md`
  (9 files: drop `role:` line).
- **Docs:** `docs/guides/configuration.md` updated; an ADR recorded in
  `docs/adr/` (staged Proposed in this change, numbered at archive).
- **Tests:** `config_test.go`, `agentfm` tests, adapter tests rewritten to the
  new shape; new must-declare error cases added.
- **Users:** anyone who relied on `[models.claude.architectural]` to pin their
  main model via homonto must move it to `[settings.claude].model`. No shim.
