# Next Agent Handoff

This file is the first stop for future agents. It summarizes the current
project state, the verified checks from the last deep audit, and the highest
value work left to do. Treat older review files as historical unless they agree
with this handoff and the current source.

## Current Verified State

- `rtk go test ./...` passed: 129 tests in 15 packages.
- `rtk go vet ./...` passed with no issues.
- `rtk go build -o /tmp/opencode/homonto-analysis-build .` succeeded.
- `rtk go test -race ./...` passed: 129 tests in 15 packages.
- `rtk gofmt -l .` clean. `homonto version` prints the stamped version.

## Fixed Since The Original Deep Review

- Claude MCP projection now uses the real schema: `command` string plus `args`.
- Claude import now preserves `command` plus `args` instead of dropping args.
- Missing-state old values are redacted instead of printed.
- State-recorded pruning exists for MCPs, settings, plugins, and skills.
- JSON path segments are escaped for dotted and special names.
- Skill path traversal is rejected by config validation.
- Atomic writes preserve existing modes and create new files as `0600`.
- Cross-adapter partial apply persists each successful adapter's state before a
  later adapter can fail.
- Plan output is sorted for deterministic rendering.
- Non-object JSON roots are rejected before writes.
- **State adoption:** a declared value already matching disk is adopted into
  state silently via an `adopt` action (no file rewrite), so pruning and drift
  see pre-existing matching resources. See `internal/adapter/{claude,opencode}`
  and `internal/plan` (`HasAdoptions`).
- **True drift in `status`:** `engine.Status` compares each adapter's
  `ObserveHashes` (hash of current on-disk value) against the recorded
  `Entry.Applied`, separate from pending desired-vs-disk config changes; drifted
  keys are excluded from the pending count. A pure `homonto.toml` edit is no
  longer mistaken for disk drift.
- **Input validation:** `config.Load` rejects unknown MCP targets, empty MCP
  commands, reserved settings keys (`enabledPlugins`, `mcp`, `plugin`), and
  index-like/empty managed names.
- **Skills-only apply is link-only:** adapters write a tool JSON file only when
  a managed key in it changed (`*Changed` guards); `adopt`/`noop`/`skill.*`
  leave JSON byte-for-byte untouched, so OpenCode JSONC comments survive
  link-only applies.
- **Doctor parity:** `doctor` verifies both `~/.claude/skills/<name>` and
  `~/.config/opencode/skills/<name>` links per owned skill.
- **CI expanded:** the pipeline runs gofmt, `go mod tidy -diff`, vet, build,
  test, race, a stamped-`--version` smoke, and a temp-HOME CLI smoke.

## Current Remaining Work

The original "seven highest-priority gaps" are all resolved in source except
where a gap was consciously accepted as a limitation. What genuinely remains:

1. **`import` is narrow by design.** It reads Claude global MCP servers only,
   redacts env values only, and preserves command/args verbatim. This is now
   documented in `cli-commands.md` and the README. Expand scope (OpenCode,
   settings/plugins/skills, non-stdio servers) or redact secret-looking command
   args only if a fuller migration tool is wanted.
2. **OpenCode JSONC comments are stripped** whenever an apply touches
   `opencode.jsonc`. Whole-file comment removal is an accepted, documented
   limitation; preserving comments is an open question, not a bug.
3. **Roadmap v1.1+ not started:** built-in templates, plugin configuration,
   TUI settings, and agent lifecycle (see `docs/roadmap.md`).

## Recommended Next Steps

1. Keep `NEXT_AGENT.md` synchronized with source after each behavioral change —
   the previous revision lagged behind six already-merged fixes.
2. Decide whether `import` grows into a full migration tool or stays a Claude
   MCP bootstrap; if it stays narrow, no code work is needed.
3. Pick a v1.1+ roadmap item (templates is the natural first step) and open an
   onto change workspace for it.

## Documentation Rules For Future Changes

- Living specs in `docs/specs/` must describe current behavior, not aspirations.
- Put planned work in `docs/roadmap.md` or a change workspace, not as false SHALLs.
- If a review is historical, say so at the top or rewrite it as a current-state
  handoff.
- When code and docs disagree, source wins until a verified code change lands.
- Keep README claims conservative: mention known limitations where users can hit
  them immediately.
