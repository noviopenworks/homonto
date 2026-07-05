# Verification Report: validate-config

- **Date:** 2026-07-05
- **Mode:** light (why: `workflow: fix`, not upgraded — a bounded config-validation fix in one module)
- **Range:** 01f2092..HEAD on `fix/20260705/validate-config`
- **Result: pass**

## Scenario evidence

The bug's reproduction is the core scenario: the three invalid inputs that
previously loaded silently now fail fast. Evidence is fresh from this round —
the real binary (`go build -o /tmp/hv .`) run against scratch configs, plus the
delta-spec scenarios' tests.

| Requirement / Scenario | Verdict | Evidence (fresh command + output) |
|---|---|---|
| config-model: Unknown target is rejected | pass | `plan` on `targets=["claud"]` → `error: parse config: mcps entry "x" targets unknown tool "claud"; valid targets are "claude" and "opencode"`. Test `TestLoadRejectsUnknownTargets` PASS (incl. `""`, `"vscode"`, mixed good/bad) |
| config-model: MCP without a command is rejected | pass | `plan` on an MCP with no `command` → `error: parse config: mcps entry "foo" has no command; an MCP server needs a command to run`. Test `TestLoadRejectsEmptyCommand` PASS (missing and `command=[]`) |
| config-model: Reserved settings key is rejected | pass | `plan` on `settings.claude.enabledPlugins` → `error: ... key "enabledPlugins" is reserved ...`; on `settings.opencode.mcp` → `error: ... key "mcp" is reserved ...`. Test `TestLoadRejectsReservedSettingKeys` PASS (all three) |
| config-model: No targets / explicit targets honored; non-colliding keys load | pass | Valid config (`targets=["claude","opencode"]`, `settings.claude.model`) plans normally: `claude: + mcp.ok = {...}`. `TestLoad` + good-case loops PASS |

## Design conformance

Preset fix — no design.md. The implementation matches the proposal's fix scope:
all three checks live in `config.Load` (`internal/config/config.go`), fail fast,
and name the offending entry. The reserved set is complete for real *write*
collisions (claude writes only `enabledPlugins` into `settings.json`; opencode
writes only `mcp`/`plugin` top-level into `opencode.jsonc`), confirmed by the
skeptic against both adapters' projection.

## Adversarial pass

One skeptic (light mode). Verdict: **safe to close** — could not produce a false
negative (unknown-target edge cases `""`, whitespace, case, mixed lists all
rejected; reserved-key case/substring variants correctly allowed since JSON keys
are case-sensitive; reserved set complete) nor a false positive (repo's own
`homonto.toml`, the sample config, `targets=[]`, and all non-colliding examples
still load). `go test ./...` and `-race` clean (128).

One MINOR finding, triaged and addressed: the fix's test/spec had blessed
`settings.claude.mcpServers` as a "harmless" non-colliding key, but claude's
`current()` skips reading it back from `settings.json`, so it is non-idempotent
at apply. Per user decision, the key stays unreserved (it is not a *write*
collision) but the misleading "harmless" test case was removed (commit 22d5106);
the underlying adapter non-idempotency is recorded as a deviation/follow-up.

## Regression

Fresh, this round:

- `go build ./...` → Success
- `go vet ./...` → No issues found
- `go test ./...` → 128 passed in 15 packages
- `go test -race ./...` → 128 passed in 15 packages
- `gofmt -l internal/` → (empty — all formatted)

## Deviations

1. **`settings.claude.mcpServers` non-idempotency (pre-existing, out of scope).**
   A `mcpServers` key under `[settings.claude]` loads (it is not a write
   collision — MCPs live in `.claude.json`) but re-creates on every apply
   because claude's `current()` (`internal/adapter/claude/claude.go`) skips
   `mcpServers`/`enabledPlugins` when reading `settings.json`. Accepted by the
   user as unreserved; the fix no longer blesses it. Candidate follow-up: make
   claude's `current()` read those keys back, or reserve `mcpServers` under
   `settings.claude`. Not a regression from this change.
