# Brainstorm Summary

- Change: remove-model-tiers
- Date: 2026-07-21

## Confirmed Technical Approach

Remove the model tier system entirely (D1): delete `role:` frontmatter,
`[models.<tool>.<tier>]` blocks, `agentfm.TierNames`/`Tiers`/`Roles`, and the
all-tiers-required `validateModels`. Subagent models centralized in
`homonto.toml` `[subagents.<name>.<tool>]` (D2). Homonto stops managing the main
session model — delete `routeModelSetting` and the model half of `routeSettings`
(D3). Add a must-declare check: every declared subagent needs a per-tool model
or load fails naming the offender (D4). Drop the redundant
`[subagents.onto-reviewer]`/`[subagents.onto-explorer]` source blocks (D5).
Operator-confirmed in brainstorming; no open clarifying questions.

## Key Trade-offs and Risks

- 18 explicit `[subagents.<name>.<tool>]` blocks in homonto.toml (verbosity
  accepted for explicitness).
- Operators who pinned main model via `[models.claude.architectural]` must move
  it to `[settings.claude].model` (documented; no shim).
- New drift class (framework adds agent, config forgets model) is caught at load
  by D4 with a precise error — strictly better than today's generic tier error.

## Testing Strategy

Rewrite tier-requiring fixtures in `config_test.go` to per-agent model shape;
add must-declare + legacy-tier-rejected error cases. Rewrite `agentfm` role
tests to explicit-model + no-model-declared cases. Adapter tests assert no main
model projected when `[settings.*]` absent; explicit settings still project.
`go build ./...`, `go vet ./...`, `go test ./internal/...`, `homonto doctor`
must all pass.

## Spec Patches

None. The `agent-models` delta spec (specs/agent-models/spec.md) already carries
the four requirements with WHEN/THEN scenarios; no supplementing needed.
