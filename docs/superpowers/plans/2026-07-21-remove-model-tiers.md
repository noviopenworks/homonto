---
change: remove-model-tiers
design-doc: docs/superpowers/specs/2026-07-21-remove-model-tiers-design.md
base-ref: f77dbf105a29aec9cfa8984637afbb14b9a9c82a
---

# Implementation Plan: remove-model-tiers

Execution sequence for the build phase. Per-file detail (exact deletions, the
`validateModels` rewrite, the target `homonto.toml`, test mappings) lives in
the Design Doc (`design-doc` above); `openspec/changes/remove-model-tiers/tasks.md`
holds the requirement-grouped checkbox tracker. This plan is the
compile-coupled execution order.

## Coupling note

Steps 1–3 are mutually coupled — removing `Role`/`TierNames` (agentfm) breaks
callers in config and adapters. They must land as one compiling unit. Step 4
(frontmatter) and 5 (homonto.toml) are safe after that. The tree does not
compile between steps 1 and 3; that is expected and called out in the Design
Doc.

## Execution order

1. **agentfm** (`internal/agentfm/agentfm.go`) — delete `Role`, `TierNames`,
   `Tiers`, `Roles`; rewrite `specFor(name)` (override-only, error on empty
   model); drop the unknown-role error; fix the package doc comment.
2. **config** (`internal/config/{config.go,load.go,validate.go}`) — remove
   `Config.Models` + the tier-map trim (`load.go:87-91`); rewrite
   `validateModels` to the must-declare per-subagent check (reuse
   `validateModelSpec` with `requireModel=true`); keep `validateSubagentOverrides`.
3. **adapters** (`internal/adapter/{claude,opencode}`) — delete
   `routeModelSetting` (claude) and the model derivation in `routeSettings`
   (opencode); explicit `[settings.*]` projection unchanged.
   → `go build ./...` must pass here (tree compiles again).
4. **catalog frontmatter** (`catalog/subagents/*.md`, 9 files) — delete the
   `role:` line from each; leave `read_only`/`dialogs`/`spawn`.
5. **homonto.toml** — delete 6 `[models.*.*]` + 2 redundant `[subagents.*]`
   source blocks; add 18 `[subagents.<name>.<tool>]` model blocks per the
   Design Doc's target listing.
6. **scaffold + docs** — replace tier examples in `internal/scaffold/scaffold.go`
   (L55-76); rewrite `docs/guides/configuration.md` models section.
7. **tests** — rewrite `config_test.go` tier fixtures (L64/76/368/387/577-592)
   to per-agent shape; add must-declare + legacy-tier-rejected cases; rewrite
   `agentfm` role tests; adapter tests assert no main model projected.
8. **verify** — `go build ./...`, `go vet ./...`,
   `go test ./internal/agentfm/... ./internal/config/... ./internal/adapter/... ./internal/scaffold/...`,
   `homonto doctor` (exit 0); grep residual `TierNames`/`role:`/`[models.`.

## Verification anchors

- The spec's four requirements (`specs/agent-models/spec.md`) map to: step 5+2
  (every agent resolves an explicit model), step 1+2 (no tier/role), step 3
  (main model unmanaged), step 2 (`validateModelSpec` preserved).
- `homonto doctor` is the headline behavior fix: red → green.
