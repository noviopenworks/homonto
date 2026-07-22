---
comet_change: remove-model-tiers
role: technical-design
canonical_spec: openspec
---

# Design Doc: remove-model-tiers (deep technical refinement)

This doc deepens the open-phase `design.md` (decisions D1–D5) into a
file-by-file build plan. The delta spec `specs/agent-models/spec.md` remains
canonical for requirements; this doc is the implementation guide for the build
phase.

## Current shape (what's there now)

- `internal/agentfm/agentfm.go`: `TierNames` (L53), `Tiers` set (L57), the
  frontmatter `Role string` field, `RenderContext.Roles map[string]ModelSpec`
  (L108), `specFor(name, role)` → `Roles[role].merge(Overrides[name])` (L114),
  and the `unknown role` error at L202.
- `internal/config/validate.go`: `validateModels` (L459) — unknown-tier reject
  loop (L464–477) + all-tiers-required loop over `agentfm.TierNames`
  (L479–484); `validateModelSpec` (L426) for effort/variant; plus
  `validateSubagentOverrides` for `[subagents.<name>.<tool>]`.
- `internal/config/load.go`: L79–85 already builds `Subagents[name].Claude` /
  `.OpenCode` per-tool routes; L87–91 trims the tier maps (delete).
- `internal/config/config.go`: `Config.Models.{Claude,OpenCode}` tier maps.
- `internal/adapter/claude/claude.go`: `routeModelSetting` (L210) reads
  `Models.Claude["architectural"]`; called by `desiredProjectSettings` (L224).
- `internal/adapter/opencode/opencode.go`: `routeSettings` (L169) reads
  `Models.OpenCode["architectural"]` (`model`) + `["trivial"]` (`small_model`);
  called by `desiredSettings` (L186).
- `internal/scaffold/scaffold.go`: L55–76 commented tier examples.
- `catalog/subagents/*.md` (9): each has a `homonto: role:` line.
- `homonto.toml`: 6 `[models.*.*]` blocks + redundant
  `[subagents.onto-reviewer]` / `[subagents.onto-explorer]` source blocks.

## File-by-file changes

### `internal/agentfm/agentfm.go`

1. Delete `TierNames`, `Tiers`, the `Role` field of the homonto frontmatter
   struct, and `RenderContext.Roles`.
2. Replace `specFor(name, role)` with `specFor(name) ModelSpec` that returns
   `Overrides[name]`. The caller (render) errors if the returned `ModelSpec` has
   an empty `Model`: `agentfm: agent %q has no model for tool %s; declare
   [subagents.%s.%s] in homonto.toml`. (This is the render-side backstop; the
   primary enforcement is the load-time check below.)
3. Delete the `unknown role` error (L202) and any role-validation branch.
4. Update the package doc comment (the `role: review` example at the top) to
   show the explicit `[subagents.<name>.<tool>]` model instead.

### `internal/config/config.go` + `load.go`

1. Remove `Config.Models` and its `ModelConfig`/`ModelRoute` map wiring that
   exists only to hold tiers. Keep `ModelRoute` (model + effort + variant) — it
   still types the per-subagent `Claude`/`OpenCode` override fields.
2. Delete the tier-map trim loop at `load.go:87–91`.

### `internal/config/validate.go`

Rewrite `validateModels` to the must-declare check (D4):

```go
func validateModels(c *Config) error {
    for _, tool := range c.EnabledModelTools() {
        for _, name := range sortedSubagentNames(c) { // deterministic order
            r := subagentRouteFor(c, name, tool) // Subagents[name].Claude/.OpenCode
            label := "subagents." + name + "." + tool
            if err := validateModelSpec(tool, label, r, /*requireModel=*/ true); err != nil {
                return err
            }
        }
    }
    return validateSubagentOverrides(c) // unchanged: effort/variant per-tool checks
}
```

The `requireModel=true` path already emits `parse config: %s model is required`
(`validateModelSpec` L430–432), so a missing `[subagents.<name>.<tool>]` model
fails with `parse config: subagents.<name>.<tool> model is required` — exactly
the spec scenario.

Delete: the unknown-tier loop (L464–477) and the all-tiers-required loop
(L479–484). There is no longer any concept of a tier name to validate against.

### `internal/adapter/claude/claude.go`

Delete `routeModelSetting` (L210) and remove its caller's model key from
`desiredProjectSettings` (L224–233 becomes just the marketplace/other keys, or
returns empty if model was its only entry). The explicit
`[settings.claude].model` path is projected by the settings machinery and is
unchanged.

### `internal/adapter/opencode/opencode.go`

In `routeSettings` (L169), delete the model/small_model derivation (the
`for settingKey, level := range ...` block). `[settings.opencode]` explicit
projection in `desiredSettings` (L186) stays.

### `catalog/subagents/*.md` (9 files)

Delete the single `  role: <tier>` line from each frontmatter. Leave
`read_only`, `dialogs`, `spawn` untouched. Files: `onto.md`, `onto-explorer.md`,
`onto-reviewer.md`, `onto-implementer.md`, `onto-skeptic.md`, `to-explorer.md`,
`to-implementer.md`, `to-reviewer.md`, `to-skeptic.md`.

### `homonto.toml` (target)

Delete the 6 `[models.*.*]` blocks and the 2 redundant `[subagents.*]` source
blocks. Add 18 `[subagents.<name>.<tool>]` blocks preserving today's tier→model
intent (review = the judgment tier reviewer/skeptic used; maps to opus):

```toml
# onto framework
[subagents.onto.claude]            # dispatcher (was architectural)
model = "opus"
[subagents.onto.opencode]
model = "anthropic/claude-opus-4-8"

[subagents.onto-reviewer.claude]   # was review
model = "opus"
[subagents.onto-reviewer.opencode]
model = "anthropic/claude-opus-4-8"

[subagents.onto-skeptic.claude]    # was review
model = "opus"
[subagents.onto-skeptic.opencode]
model = "anthropic/claude-opus-4-8"

[subagents.onto-implementer.claude]  # was coding
model = "sonnet"
effort = "medium"
[subagents.onto-implementer.opencode]
model = "anthropic/claude-sonnet-4"

[subagents.onto-explorer.claude]   # was trivial
model = "haiku"
effort = "low"
[subagents.onto-explorer.opencode]
model = "openai/gpt-5-mini"
variant = "cheap"

# to framework (mirror onto's worker roles)
[subagents.to-reviewer.claude]
model = "opus"
[subagents.to-reviewer.opencode]
model = "anthropic/claude-opus-4-8"

[subagents.to-skeptic.claude]
model = "opus"
[subagents.to-skeptic.opencode]
model = "anthropic/claude-opus-4-8"

[subagents.to-implementer.claude]
model = "sonnet"
effort = "medium"
[subagents.to-implementer.opencode]
model = "anthropic/claude-sonnet-4"

[subagents.to-explorer.claude]
model = "haiku"
effort = "low"
[subagents.to-explorer.opencode]
model = "openai/gpt-5-mini"
variant = "cheap"
```

(onto has a dispatcher `onto`; `to` has none — confirmed: no `to.md` in
`catalog/subagents/`.)

### `internal/scaffold/scaffold.go`

Replace the L55–76 commented tier examples with a `[subagents.<name>.<tool>]`
example and note that the main model is optional via `[settings.<tool>]`.

### `docs/guides/configuration.md`

Rewrite the models section: no tiers; subagent models in
`[subagents.<name>.<tool>]`; main model optional via `[settings.<tool>]`.

## Edge cases & decisions

- **`EnabledModelTools()`**: unchanged. It already determines which tools
  require models per declared subagent; the must-declare check rides on it.
- **Override-only fields**: `[subagents.<name>.<tool>]` with `variant` alone but
  no `model` is invalid under the new check (model required) — correct, since
  there's no tier default for variant to merge into. `validateModelSpec`'s
  variant-needs-alias rule (L440) still fires when a model is present.
- **Subagents with `targets` restricting tools**: the must-declare check uses
  each subagent's enabled tools (via the existing per-subagent target logic), so
  an agent targeted to claude only needs a claude block. Confirm during build by
  reading `validateSubagentOverrides`'s target handling.
- **Legacy `[models.*.*]` in a config**: now an unknown key. TOML decode of
  `Config.Models` is gone, so such a block would either error at decode
  (unknown field) or be silently ignored depending on decoder strictness —
  verify and, if silently ignored, add an explicit reject (the spec scenario
  "Legacy tier block rejected" requires an error naming the key).

## Test mappings (tasks §7)

- `config_test.go` L64, 76, 368, 387, 577–592: replace the 4-tier fixtures with
  per-agent `[subagents.<name>.<tool>]` fixtures. Add: (a) missing-model error
  case asserting `subagents.<name>.<tool> model is required`; (b) legacy
  `[models.claude.architectural]` rejected case.
- `agentfm` tests: role-default → explicit-model; add no-model-declared render
  error.
- Adapter tests: assert no `model`/`small_model` key projected when
  `[settings.*]` absent; assert `[settings.claude].model` still projects.
- Doctor / `homonto-expanded` E2E: update if it references tiers or
  `[models.*.*]`.

## Build order (low-risk, each step leaves the tree compiling)

1. `agentfm` — drop Role/TierNames/Tiers/Roles, rewrite specFor. (Tree
   won't compile until step 2/3 because callers pass role; that's expected —
   steps 1–3 land together.)
2. `config` — drop Models, rewrite validateModels.
3. adapters — delete model projection.
4. `catalog/subagents/*.md` — drop role: lines (9).
5. `homonto.toml` — rewrite.
6. `scaffold.go` + `docs/guides/configuration.md`.
7. Tests.
8. Verify: `go build ./...`, `go vet ./...`, `go test ./internal/...`,
   `homonto doctor` (exit 0), grep for residual `TierNames`/`role:`/`[models.`.

## Rollback

Single revert; no in-place downgrade (old config + new code is invalid both
ways).

## Implementation Divergence

**D5 (drop the redundant `[subagents.*]` source blocks) was not applied.**

D5 assumed the onto framework declares `onto-reviewer` / `onto-explorer`, so
the top-level source blocks in `homonto.toml` were a colliding re-declaration.
That premise does not hold in this repo: `homonto.toml` declares
`frameworks.comet`, and comet was removed from the catalog (ADR 0015). No
framework declares those two agents here, so deleting the source blocks would
delete the agents rather than deduplicate them.

Consequences:

- The two `[subagents.onto-reviewer]` / `[subagents.onto-explorer]` source
  blocks remain.
- `homonto.toml` carries 4 model blocks (2 agents × 2 tools), not the 18 the
  design projected. The 9×2 shape ships in the scaffold template instead
  (`internal/scaffold/scaffold.go`), which is what new installs see.
- Nothing in the delta spec (`specs/agent-models/spec.md`) depends on D5 — the
  must-declare requirement is satisfied for exactly the agents this config
  declares, and `homonto doctor` exits 0.

The underlying question — that this repo's own config still names a framework
the catalog no longer ships — is the repo-hygiene follow-up already noted under
Risks, not a defect of this change.
