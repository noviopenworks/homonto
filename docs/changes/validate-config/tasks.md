# Tasks: validate-config

## 1. Reproduce

- [x] 1.1 Failing tests first: a config with an unknown MCP target, one with an
      empty command, and one each for reserved `settings.claude.enabledPlugins`
      / `settings.opencode.mcp` / `settings.opencode.plugin` all currently load
      without error (demonstrating the silent-accept bug).

## 2. Fix

- [x] 2.1 In `config.Load`, validate MCP target names ∈ {claude, opencode};
      fail fast naming the unknown target and the valid set.
- [x] 2.2 In `config.Load`, reject an MCP with an empty/missing command; fail
      fast naming the MCP.
- [x] 2.3 In `config.Load`, reject reserved settings keys
      (`settings.claude.enabledPlugins`, `settings.opencode.mcp`,
      `settings.opencode.plugin`); fail fast naming the key.

## 3. Spec + regression

- [ ] 3.1 Delta spec `specs/config-model.md` — ADDED input-validation
      requirements with scenarios for each rule.
- [ ] 3.2 Regression: a valid multi-target, multi-setting, multi-plugin config
      still loads; `go test ./...`, `go vet ./...`, `go build`, `go test -race`
      all pass.
