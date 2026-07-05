# Next Agent Handoff

This file is the first stop for future agents. It summarizes the current
project state, the verified checks from the last deep audit, and the highest
value work left to do. Treat older review files as historical unless they agree
with this handoff and the current source.

## Current Verified State

- `rtk go test ./...` passed: 92 tests in 15 packages.
- `rtk go vet ./...` passed with no issues.
- `rtk go build -o /tmp/opencode/homonto-analysis-build .` succeeded.
- `rtk go test -race ./...` passed: 92 tests in 15 packages.

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

## Current Highest-Priority Gaps

1. **State adoption for existing matching resources.** If disk already matches a
   non-secret desired value, adapters emit `noop` and skip state writes. Imported
   or manually pre-existing resources can therefore look managed while remaining
   invisible to pruning and some drift checks.
2. **`status` is not true disk-vs-state drift.** `engine.Drift` reuses the
   current desired `Plan()`, so config edits can appear as drift even when disk
   still matches the last apply.
3. **Invalid targets and empty commands are silently ignored.** Validate target
   names (`claude`, `opencode`) and fail fast on MCPs that cannot project.
4. **Import is partial.** It reads Claude global MCP servers only, does not import
   OpenCode/settings/plugins/skills, and redacts only env values, not secrets
   passed in command arguments.
5. **Skills-only apply still touches JSON config files.** The adapters currently
   read and write their config files even when only skill links are pending,
   which can create files or normalize OpenCode JSONC comments.
6. **`doctor` checks only Claude skill links.** It verifies content and the
   Claude link, but not the OpenCode skill symlink.
7. **CI is too narrow.** It runs vet and tests only. Add build, stamped-version
   smoke, `gofmt`, `go mod tidy -diff`, race tests, and CLI smoke coverage.

## Recommended Fix Order

1. Add state adoption for existing matching values and plugins/links.
2. Redesign `status` to compare current disk against `.homonto/state.json` rather
   than current desired config.
3. Validate target names, empty commands, and reserved settings namespaces.
4. Expand or explicitly constrain import; redact secrets in command args if they
   remain importable.
5. Avoid JSON writes for link-only changes, or document that side effect
   everywhere.
6. Add OpenCode skill-link checks to `doctor`.
7. Expand CI and release smoke checks.
8. Add a core user guide for homonto usage; the current guide coverage is mostly
   onto workflow documentation.

## Documentation Rules For Future Changes

- Living specs in `docs/specs/` must describe current behavior, not aspirations.
- Put planned work in `docs/roadmap.md` or a change workspace, not as false SHALLs.
- If a review is historical, say so at the top or rewrite it as a current-state
  handoff.
- When code and docs disagree, source wins until a verified code change lands.
- Keep README claims conservative: mention known limitations where users can hit
  them immediately.
