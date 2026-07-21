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
load, every declared subagent must resolve a model for each enabled tool:
`parse config: subagents.<name>.<tool> model is required`. *Why:* centralized
models (D2) would otherwise let an agent render with no model (silent
breakage). The check is the direct analog of today's all-tiers requirement,
but it names the offender (better than today's `models.claude.review is
required`, which does not say which agent needs it). Main has no such check
(D3 makes the main model optional).

**D5 — Drop the redundant `[subagents.*]` source blocks in `homonto.toml`.**
The onto framework already declares `onto-reviewer` / `onto-explorer` (its
`framework.toml [subagents]`); the top-level `[subagents.onto-reviewer]` /
`[subagents.onto-explorer]` source blocks in `homonto.toml:9-14` collide with
that declaration. D2's model blocks replace them.

**Alternatives considered:**
- *Keep tier blocks for main only* — leaves `architectural`/`trivial` as zombie
  names; rejected for the same coupling reason as D1.
- *New `[main.<tool>]` block* — adds a concept to replace one being removed;
  rejected (D3).
- *Models co-located in framework `.md` frontmatter* — best kills the drift
  class, but rejected by the operator's centralization preference (D2).
- *`role:` kept as an inert label* — dead indirection; rejected.

## Risks / Trade-offs

- **[Operator forgets a model block after adding a framework agent] →** D4
  fails config load with a named error before any agent renders. The failure
  mode is strictly better than today's (precise vs. generic).
- **[Operators who pinned the main model via `[models.claude.architectural]`]**
  → documented migration in the ADR: move it to `[settings.claude].model`. No
  shim, consistent with the repo's established "breaking, no shim" pattern.
- **[Verbose homonto.toml — 18 subagent blocks] →** accepted as the explicit
  cost of centralization (D2). Mitigated by clear `[subagents.<name>.<tool>]`
  naming and the scaffold template.
- **[Comet/openspec re-introduced into a repo that cleared it (commit 652db64)]**
  → operator-confirmed at change open; out of scope for this design to
  reconcile, but noted as a repo-hygiene follow-up.

## Migration Plan

1. Land code changes (D1, D3, D4) and the rewritten `homonto.toml` (D2, D5) in
   one change — there is no intermediate consistent state (old config + new
   code, or vice versa, both fail).
2. Rewrite tests to the new shape; add the two must-declare error cases.
3. Update `docs/guides/configuration.md` and `internal/scaffold` examples.
4. Record ADR in `docs/adr/` (Proposed, staged in this change; numbered at
   archive per the ADR staging rule).
5. Rollback: revert the commit; there is no in-place downgrade path.

## Open Questions

None blocking. (The comet/openspec re-introduction noted under Risks is a
separate repo-hygiene question, not a design unknown for *this* change.)
