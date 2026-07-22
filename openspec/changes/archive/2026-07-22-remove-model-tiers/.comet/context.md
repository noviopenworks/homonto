# Comet Design Handoff

- Change: remove-model-tiers
- Phase: design
- Mode: compact
- Context hash: 723df7b18b4e2dbe07f8b55bf2af6a74f9b766ec493622e110a114427882c6c2

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/remove-model-tiers/proposal.md

- Source: openspec/changes/remove-model-tiers/proposal.md
- Lines: 1-60
- SHA256: 3e56f8d85350cfb962c71bdf0788c2e9409c5564fee11d754dc1eae518cd1c22

```md
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

```

## openspec/changes/remove-model-tiers/design.md

- Source: openspec/changes/remove-model-tiers/design.md
- Lines: 1-133
- SHA256: 2824f980c44f35ec5d6de6958c0d62bd3cb39350a752e29ea48a2a1b934221a0

[TRUNCATED]

```md
## Context

Homonto configures AI-tool agents (Claude, OpenCode). Today it routes models
through a tier indirection: each subagent's frontmatter declares
`role: architectural|coding|review|trivial`; the role selects a
`[models.<tool>.<tier>]` block; `agentfm.specFor` merges that with any
`[subagents.<name>.<tool>]` override; and `validateModels`
(`internal/config/validate.go:459`) requires every enabled tool to declare all
four tiers (`agentfm.TierNames`). The tier map is *also* read directly by the
adapters for the **main** session model — Claude reads
`[models.claude.architectural]` (`adapter/claude/claude.go:214`), OpenCode
reads `[models.opencode.architectural]` (`model`) and
`[models.opencode.trivial]` (`small_model`) (`adapter/opencode/opencode.go:175`).

This coupling is brittle: adding the `review` tier in code made
`homonto doctor` fail on the shipped config, because `homonto.toml` had not
declared the new tier. The indirection's cost (a config can be invalid against
current code for reasons the config cannot see) now exceeds its benefit (one
place to change a model class).

Relevant code: `internal/agentfm` (role/tier resolution + rendering),
`internal/config` (`Models`, `validateModels`, `validateSubagentOverrides`),
`internal/adapter/{claude,opencode}` (main-model projection),
`internal/scaffold` (template examples), `catalog/subagents/*.md` (9 agent
frontmatters), `homonto.toml` (the shipped config). ADRs 0014 (adapter
contract) and 0015 (ship only onto/to frameworks) frame the surface.

## Goals / Non-Goals

**Goals:**
- Remove the tier/role indirection so an agent's model is declared where the
  agent is configured, explicitly, with no shared mutable default that code
  edits can silently invalidate.
- Make `homonto doctor` pass on the shipped config with no "missing tier"
  failure class possible in the future.
- Keep the existing override mechanism (`[subagents.<name>.<tool>]`) as the
  single source of subagent models.
- Preserve framework self-containment for agent **existence** (framework.toml
  still declares which agents a framework ships).

**Non-Goals:**
- No change to the `read_only` / `dialogs` / `spawn` frontmatter fields — those
  express real capability intent, not model tiers.
- No change to framework install/declaration mechanics (`framework.toml
  [subagents]`, dependency resolution, materialization).
- No change to the comet/openspec skills (upstream artifacts homonto does not
  author — ADR 0015).
- No shim / automatic migration of old `[models.*.*]` configs.

## Decisions

**D1 — Remove the tier system entirely.** Delete `role:` frontmatter, the
`[models.<tool>.<tier>]` config blocks, `agentfm.TierNames` / `Tiers` /
`Roles`, and the all-tiers-required branch of `validateModels`. *Why whole
removal over "role only, keep tier blocks as optional defaults":* the breakage
showed any surviving tier name is a latent collision point between code and
config; half-measures leave the class of bug alive. The override mechanism
already exists as a complete replacement.

**D2 — Subagent models live in `homonto.toml` `[subagents.<name>.<tool>]`.**
Existence still comes from the onto/to framework declarations; models are
centralized per-agent per-tool in the installer config. *Why centralized over
co-located in framework `.md` frontmatter:* the user's explicit choice — keep
all model selection in one place the operator owns. *Trade-off accepted:* a
framework that ships a new agent now requires every installer to add a model
block (mitigated by D4's loud, named error). The existing
`Subagents[name].Claude` / `.OpenCode` fields (`config/load.go:79-85`) already
hold this shape, so no new data model is needed.

**D3 — Homonto stops managing the main session model.** Delete
`routeModelSetting` (Claude) and the route-derived model half of `routeSettings`
(OpenCode). The main model is whatever the tool itself defaults to; the
existing explicit `[settings.claude].model` / `[settings.opencode].model` /
`.small_model` path remains for operators who want homonto to set it. *Why
unmanaged over a new `[main.<tool>]` block:* the main model is not homonto's
concern — it is the operator's tool default. Removing the projection also
removes the last reader of the tier map, which is what enables D1. No
capability is lost: explicit settings still work.

**D4 — Must-declare validation replaces the tier requirement.** At config

```

Full source: openspec/changes/remove-model-tiers/design.md

## openspec/changes/remove-model-tiers/tasks.md

- Source: openspec/changes/remove-model-tiers/tasks.md
- Lines: 1-85
- SHA256: dc41a45b3743afac0585b60058bbedc20a292be898dfc0240fa2777f1dfccf5b

[TRUNCATED]

```md
## 1. Remove the tier system from `agentfm`

- [ ] 1.1 Delete the `Role` field from the `homonto` frontmatter struct
  (`internal/agentfm/agentfm.go`); drop the `unknown role` error path.
- [ ] 1.2 Delete `TierNames`, `Tiers`, and `Roles` (the role→spec map in
  `RenderContext`).
- [ ] 1.3 Rewrite `specFor` to read only `Overrides[name]`; it returns the
  declared per-tool spec or signals "no model" so the caller errors.
- [ ] 1.4 Update `agentfm` doc comments (the `role:` example at the top of
  `agentfm.go`) to the explicit-model model.

## 2. Remove `Models` from config; add must-declare validation

- [ ] 2.1 Remove `Config.Models` (the `Claude`/`OpenCode` tier maps) and the
  `ModelConfig`/`ModelRoute` tier wiring that only served tiers; keep
  `ModelRoute` where the per-subagent override still uses it.
- [ ] 2.2 Drop the tier-map trimming in `config/load.go:87`.
- [ ] 2.3 Rewrite `validateModels` (`internal/config/validate.go:459`): remove
  the unknown-tier check and the all-tiers-required loop; replace with a
  per-subagent must-declare check — every declared subagent must have a
  `[subagents.<name>.<tool>]` model for each enabled tool, else
  `parse config: subagents.<name>.<tool> model is required`.
- [ ] 2.4 Keep `validateModelSpec` (effort/variant per-tool checks) and route it
  at the per-subagent blocks.

## 3. Stop managing the main session model

- [ ] 3.1 Delete `routeModelSetting` and its caller in `desiredProjectSettings`
  (`internal/adapter/claude/claude.go`); confirm explicit
  `[settings.claude].model` still projects via the settings path.
- [ ] 3.2 Remove the model/small_model derivation from `routeSettings`
  (`internal/adapter/opencode/opencode.go`); keep `[settings.opencode]`
  explicit projection.
- [ ] 3.3 Grep for any other reader of `c.Models.*` and remove/update; ensure
  `go vet ./...` is clean of `Models` references.

## 4. Strip `role:` from catalog subagent frontmatter

- [ ] 4.1 Delete the `role:` line from all 9 `catalog/subagents/*.md`:
  `onto`, `onto-explorer`, `onto-reviewer`, `onto-implementer`, `onto-skeptic`,
  `to-explorer`, `to-implementer`, `to-reviewer`, `to-skeptic`. Leave
  `read_only`, `dialogs`, `spawn` untouched.

## 5. Rewrite `homonto.toml`

- [ ] 5.1 Delete every `[models.*.*]` block (6 today).
- [ ] 5.2 Delete the redundant `[subagents.onto-reviewer]` and
  `[subagents.onto-explorer]` source blocks (framework owns existence).
- [ ] 5.3 Add `[subagents.<name>.<tool>]` model blocks for all 9 agents × 2
  tools (18 blocks), preserving today's model intent: dispatcher=opus,
  reviewer/skeptic=opus, implementer=sonnet(medium), explorer=haiku(low);
  OpenCode equivalents + explorer `variant="cheap"`.
- [ ] 5.4 Run `homonto doctor` — must exit 0.

## 6. Scaffold + docs

- [ ] 6.1 Replace the `[models.<tool>.<tier>]` examples in
  `internal/scaffold/scaffold.go` with `[subagents.<name>.<tool>]` examples.
- [ ] 6.2 Update `docs/guides/configuration.md` to the new shape (no tiers,
  subagent models centralized, main model optional via `[settings.*]`).
- [ ] 6.3 Draft the ADR (Proposed, no number) at
  `openspec/changes/remove-model-tiers/adr-NNNN-tier-removal.md` per the ADR
  README staging rule; it gets moved/numbered into `docs/adr/` at archive.

## 7. Tests

- [ ] 7.1 `config_test.go`: rewrite the tier-requiring fixtures (lines ~64, 76,
  368, 387, 577-592) to the `[subagents.<name>.<tool>]` shape; add the
  must-declare error case (missing `subagents.<name>.<tool> model`) and the
  legacy-`[models.*.*]`-rejected case.
- [ ] 7.2 `agentfm` tests: role-default cases become explicit-model cases; add
  the "no model declared" render failure.
- [ ] 7.3 Adapter tests: assert no main model is projected when `[settings.*]`
  is absent; assert explicit `[settings.*].model` still projects.
- [ ] 7.4 Update the doctor / E2E fixture (`homonto-expanded`) if it references
  tiers or `[models.*.*]`.

## 8. Verification

- [ ] 8.1 `go build ./...` clean; `go vet ./...` clean of `Models`/`role`

```

Full source: openspec/changes/remove-model-tiers/tasks.md

## openspec/changes/remove-model-tiers/specs/agent-models/spec.md

- Source: openspec/changes/remove-model-tiers/specs/agent-models/spec.md
- Lines: 1-85
- SHA256: cd68ecda8bb8e8b06cda0ab64d96b27850d4349cb220e6f74d6299cf8fc966d6

[TRUNCATED]

```md
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

```

Full source: openspec/changes/remove-model-tiers/specs/agent-models/spec.md
