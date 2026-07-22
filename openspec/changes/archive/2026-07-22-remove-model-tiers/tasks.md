## 1. Remove the tier system from `agentfm`

- [x] 1.1 Delete the `Role` field from the `homonto` frontmatter struct
  (`internal/agentfm/agentfm.go`); drop the `unknown role` error path.
- [x] 1.2 Delete `TierNames`, `Tiers`, and `Roles` (the role→spec map in
  `RenderContext`).
- [x] 1.3 Rewrite `specFor` to read only `Overrides[name]`; it returns the
  declared per-tool spec or signals "no model" so the caller errors.
- [x] 1.4 Update `agentfm` doc comments (the `role:` example at the top of
  `agentfm.go`) to the explicit-model model.

## 2. Remove `Models` from config; add must-declare validation

- [x] 2.1 Remove `Config.Models` (the `Claude`/`OpenCode` tier maps) and the
  `ModelConfig`/`ModelRoute` tier wiring that only served tiers; keep
  `ModelRoute` where the per-subagent override still uses it.
- [x] 2.2 Drop the tier-map trimming in `config/load.go:87`.
- [x] 2.3 Rewrite `validateModels` (`internal/config/validate.go:459`): remove
  the unknown-tier check and the all-tiers-required loop; replace with a
  per-subagent must-declare check — every declared subagent must have a
  `[subagents.<name>.<tool>]` model for each enabled tool, else
  `parse config: subagents.<name>.<tool> model is required`.
- [x] 2.4 Keep `validateModelSpec` (effort/variant per-tool checks) and route it
  at the per-subagent blocks.

## 3. Stop managing the main session model

- [x] 3.1 Delete `routeModelSetting` and its caller in `desiredProjectSettings`
  (`internal/adapter/claude/claude.go`); confirm explicit
  `[settings.claude].model` still projects via the settings path.
- [x] 3.2 Remove the model/small_model derivation from `routeSettings`
  (`internal/adapter/opencode/opencode.go`); keep `[settings.opencode]`
  explicit projection.
- [x] 3.3 Grep for any other reader of `c.Models.*` and remove/update; ensure
  `go vet ./...` is clean of `Models` references.

## 4. Strip `role:` from catalog subagent frontmatter

- [x] 4.1 Delete the `role:` line from all 9 `catalog/subagents/*.md`:
  `onto`, `onto-explorer`, `onto-reviewer`, `onto-implementer`, `onto-skeptic`,
  `to-explorer`, `to-implementer`, `to-reviewer`, `to-skeptic`. Leave
  `read_only`, `dialogs`, `spawn` untouched.

## 5. Rewrite `homonto.toml`

- [x] 5.1 Delete every `[models.*.*]` block (6 today).
- [x] 5.2 Delete the redundant `[subagents.onto-reviewer]` and
  `[subagents.onto-explorer]` source blocks (framework owns existence).
  **Deviation:** kept. This repo's config declares `frameworks.comet`, not
  `onto`, so no framework declares these agents — dropping the source blocks
  would delete the agents rather than deduplicate them.
- [x] 5.3 Add `[subagents.<name>.<tool>]` model blocks for all 9 agents × 2
  tools (18 blocks), preserving today's model intent: dispatcher=opus,
  reviewer/skeptic=opus, implementer=sonnet(medium), explorer=haiku(low);
  OpenCode equivalents + explorer `variant="cheap"`.
  **Deviation:** 4 blocks, not 18 — only the 2 agents this config actually
  declares (5.2). Model intent preserved: reviewer=opus, explorer=haiku(low) /
  `openai/gpt-5-mini` + `variant="cheap"`. The 9×2 shape ships in the scaffold
  template (6.1) instead.
- [x] 5.4 Run `homonto doctor` — must exit 0.

## 6. Scaffold + docs

- [x] 6.1 Replace the `[models.<tool>.<tier>]` examples in
  `internal/scaffold/scaffold.go` with `[subagents.<name>.<tool>]` examples.
- [x] 6.2 Update `docs/guides/configuration.md` to the new shape (no tiers,
  subagent models centralized, main model optional via `[settings.*]`).
- [x] 6.3 Draft the ADR (Proposed, no number) at
  `openspec/changes/remove-model-tiers/adr-NNNN-tier-removal.md` per the ADR
  README staging rule; it gets moved/numbered into `docs/adr/` at archive.

## 7. Tests

- [x] 7.1 `config_test.go`: rewrite the tier-requiring fixtures (lines ~64, 76,
  368, 387, 577-592) to the `[subagents.<name>.<tool>]` shape; add the
  must-declare error case (missing `subagents.<name>.<tool> model`) and the
  legacy-`[models.*.*]`-rejected case.
- [x] 7.2 `agentfm` tests: role-default cases become explicit-model cases; add
  the "no model declared" render failure.
- [x] 7.3 Adapter tests: assert no main model is projected when `[settings.*]`
  is absent; assert explicit `[settings.*].model` still projects.
- [x] 7.4 Update the doctor / E2E fixture (`homonto-expanded`) if it references
  tiers or `[models.*.*]`.

## 8. Verification

- [x] 8.1 `go build ./...` clean; `go vet ./...` clean of `Models`/`role`
  references.
- [x] 8.2 `go test ./internal/agentfm/... ./internal/config/... ./internal/adapter/... ./internal/scaffold/...` green.
- [x] 8.3 `homonto doctor` exits 0 on the rewritten `homonto.toml`.
- [x] 8.4 Grep confirms no residual `TierNames` / `role:` / `[models.` in code
  or catalog.
