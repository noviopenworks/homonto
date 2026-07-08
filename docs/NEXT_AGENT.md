# Next Agent Handoff

This file is the first stop for future agents. It summarizes the current
project state, the verified checks from the last deep audit, and the highest
value work left to do. Treat older review files as historical unless they agree
with this handoff and the current source.

## Current Verified State

Last checked locally on 2026-07-08:

- `gofmt -l .` clean.
- `go mod tidy -diff` clean.
- `go vet ./...` passed with no issues.
- `go build ./...` succeeded.
- `go test ./...` passed: 153 tests in 16 packages.
- `go test -race ./...` passed: 153 tests in 16 packages.
- `./scripts/docker-test.sh` passed.
- `go run . status` reports `No drift` (repo dogfooded at project scope).

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
  commands, reserved settings keys (claude `enabledPlugins`/`mcpServers`,
  opencode `mcp`/`plugin`), and index-like/empty managed names.
- **Skills-only apply is link-only:** adapters write a tool JSON file only when
  a managed key in it changed (`*Changed` guards); `adopt`/`noop`/`skill.*`
  leave JSON byte-for-byte untouched, so OpenCode JSONC comments survive
  link-only applies.
- **Doctor parity:** `doctor` verifies both the claude and opencode skill links
  per owned skill, at the location for the active `[skills] scope` (user home or
  project root) via `skillpath.Dir`.
- **CI expanded:** `ci.yml` runs gofmt, `go mod tidy -diff`, vet, build, test,
  race, a stamped-`--version` smoke, a temp-HOME CLI smoke, the Docker apply
  smoke, and `govulncheck`; workflows are least-privilege (`contents: read`).
- **Release plumbing (Iteration 1 closed):** `.github/workflows/release.yml`
  builds/checksums/publishes cross-platform binaries on a `v*` tag;
  `docs/release-checklist.md` documents tag/build/checksums/smoke/rollback and
  the deferred-CodeQL decision.
- **Binary-level coverage (Iteration 2 closed):** the Docker smoke now covers MCP
  + settings projection, secret env-ref resolution (resolved in files, `${ref}`
  in state, never leaked), `init`, `import`/`--force`, and real-file/foreign-
  symlink conflicts; `internal/cli/command_test.go` adds init/import/error tests.
- **Public-beta polish (Iteration 3 closed):** README leads with the user path
  and a consolidated "Known limitations" section, with internal material under
  "For contributors"; `docs/release-notes.md` carries the accepted limitations
  into every release's notes.
- **Foreign skill symlink is a conflict (Iteration 0 blocker closed):**
  `link.Link`/`link.Plan` now relink only a symlink whose target is inside the
  managed content root; a symlink pointing outside `content/` is a user-owned
  conflict and is never removed or repointed. Regression tests live at linker
  level (`internal/link/linker_test.go`) and adapter/apply level
  (`TestForeignSkillSymlinkAborts` in both adapters).
- **`settings.claude.mcpServers` is rejected (Iteration 0 blocker closed):**
  `config.Load` reserves it — claude's `current()` skips reading it back, so it
  would be non-idempotent. `TestLoadRejectsReservedSettingKeys` names the key.
- **Scope-switch status is pending, not drift (Iteration 0 blocker closed):**
  `ObserveHashes` reads each skill link at the destination state recorded (via
  `recordedDst`), not the current scope's dir, so a pending `[skills] scope`
  change shows as a pending relocation while old links are intact.
  `TestScopeSwitchStatusReportsPendingNotDrift` covers both switch directions.
- **Repo dogfooded at project scope (Iteration 0 blocker closed):**
  `homonto.toml` sets `scope = "project"`, so the onto dev skills link under this
  repo's own `.claude`/`.opencode` (gitignored) instead of the maintainer's
  global home. `apply --yes` was run and `status` reports `No drift`.

## Current Remaining Work

Every engineering item in `docs/road-to-release.md` (Iterations 0–4) is closed —
see "Fixed Since The Original Deep Review" above. What is left is inherently
maintainer-owned and cannot be done autonomously:

1. **Tag `v0.1.0-rc.1`.** All gates are green; pushing the tag triggers
   `release.yml`. Follow `docs/release-checklist.md`.
2. **Run the `go install github.com/noviopenworks/homonto@<tag>` smoke** from a
   clean environment once the tag exists (only `go install .` and an out-of-repo
   binary run could be verified without a tag).
3. **Promote to `v0.1.0`** after at least one clean dogfood cycle with the tagged
   binary.

Beyond release, the post-v1 roadmap (built-in templates, plugin configuration,
TUI settings, agent lifecycle) remains unstarted feature work. Two accepted beta
limitations are documented, not bugs: OpenCode JSONC comment loss on writes, and
`import` being a narrow Claude MCP bootstrap.

## Recommended Next Steps

1. When ready to release, work `docs/release-checklist.md` end to end and push
   `v0.1.0-rc.1`; the rest of Iteration 4 unblocks from there.
2. Keep `NEXT_AGENT.md` synchronized with source after each behavioral change.
3. After v0.1.0 ships, pick a v1.1+ roadmap item (templates is the natural first
   step) and open an onto change workspace for it.

## Documentation Rules For Future Changes

- Living specs in `docs/specs/` must describe current behavior, not aspirations.
- Put planned work in `docs/roadmap.md` or a change workspace, not as false SHALLs.
- If a review is historical, say so at the top or rewrite it as a current-state
  handoff.
- When code and docs disagree, source wins until a verified code change lands.
- Keep README claims conservative: mention known limitations where users can hit
  them immediately.
