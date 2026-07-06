# Current Deep Review — homonto + onto (2026-07-04)

> **Superseded as a current-state reference.** The "Open Findings" below were
> the state of the tree on 2026-07-04. All of them have since been resolved by
> the `state-source-of-truth` (adoption + true `status` drift + link-only JSON
> writes), `validate-config` (target/command/reserved-key validation),
> `doctor-opencode-link` (OpenCode skill-link checks), and `expand-ci` changes.
> Treat `docs/NEXT_AGENT.md` as the authoritative current handoff; read this
> file only as the 2026-07-04 point-in-time record. The test count has since
> grown from 92 to 129.

This document replaces the original harsh review with the current project state.
The original review was useful, but several blockers it named have since been
fixed. Future agents should use this file together with `docs/NEXT_AGENT.md`
and current source, not old archived change text, as their starting point.

## Verified Checks

- `rtk go test ./...` passed: 92 tests in 15 packages.
- `rtk go vet ./...` passed with no issues.
- `rtk go build -o /tmp/opencode/homonto-analysis-build .` succeeded.
- `rtk go test -race ./...` passed: 92 tests in 15 packages.

## Current Verdict

homonto is no longer in the original "core does not work" state. The Claude MCP
schema, import argument preservation, secret redaction on missing state,
state-recorded pruning, JSON path escaping, traversal validation, file mode
preservation, cross-adapter partial apply records, deterministic plan ordering,
and non-object JSON root rejection are all present in source and backed by tests.

The project is still not release-ready. The remaining issues are mostly semantic
and product-safety gaps: state adoption, true status/drift behavior, validation,
partial import, link-only side effects, thin health checks, and insufficient CI
coverage. Fix those before presenting v1 as a dependable declarative manager.

## Fixed Since The Original Review

1. **Claude MCP schema fixed.** The Claude adapter emits `type: "stdio"`,
   `command` as a string, and `args` separately. See
   `internal/adapter/claude/claude.go` and schema tests.
2. **Import preserves Claude args.** The importer reads real Claude `command` +
   `args` and tolerates the old array form.
3. **Missing-state secret leak fixed.** Unknown-provenance old values are redacted
   instead of printed.
4. **State-recorded pruning exists.** Removed MCPs/settings/plugins/skills plan
   as deletes when the key was previously recorded in state.
5. **JSON path escaping exists.** Dotted and special names are escaped for
   gjson/sjson path use.
6. **Skill traversal blocked.** `skills.own` entries must be plain directory
   names.
7. **File modes preserved.** Atomic writes preserve existing modes and create new
   managed files as `0600`.
8. **Cross-adapter partial apply improved.** State is saved after each successful
   adapter, so a later adapter failure does not erase earlier records.
9. **Plan ordering deterministic.** Adapter changes are sorted.
10. **Non-object roots rejected.** Managed JSON files must have object roots.

## Open Findings

### P0/P1 — State adoption gap

If a non-secret desired value already matches disk, adapters emit `noop` and
`Apply` skips it, so no state record is written. Existing matching MCPs,
settings, plugins, or links can therefore look up to date while remaining
unmanaged for pruning and status.

Evidence:
- `internal/adapter/claude/claude.go`: direct equality returns noop.
- `internal/adapter/opencode/opencode.go`: `planKey` returns noop on equality.
- Both adapters skip `noop` during apply.

Needed fix: when a declared key matches disk but lacks state, adopt it into state
without rewriting user files. Cover MCPs, settings, plugins, and skill links.

### P1 — `status` is not true last-applied drift

`engine.Drift` reuses the current desired `Plan()` and filters for keys already
in state. That catches some out-of-band changes, but it also reports pending
config edits as drift and cannot independently compare disk to the recorded
last-applied snapshot.

Evidence:
- `internal/engine/status.go` calls `e.Plan()`.
- Adapter planning compares current disk to current desired config.

Needed fix: add adapter support for reading current disk values by managed key and
compare those values to state entries directly. Keep desired-config changes in
`plan`, not `status`.

### P1 — Target and command validation gaps

`targets = ["claud"]` is accepted by config loading and ignored by both adapters.
Empty MCP commands are also silently skipped by adapters. With state-recorded
resources, these mistakes can look like de-declarations and may lead to pruning.

Evidence:
- `config.MCP.TargetsOrAll()` returns explicit targets without validation.
- Adapters filter by string containment and skip empty command arrays.

Needed fix: validate target names and command shape in `config.Load`. Consider
reserved top-level setting names too (`mcp`, `plugin`, `enabledPlugins`) because
settings can collide with adapter-owned namespaces.

### P1/P2 — Import remains partial

Import reads Claude global MCP servers only. It does not import OpenCode, Claude
settings/plugins/skills, project-scoped configs, or non-stdio servers. Redaction
is limited to env values; secrets embedded in command arguments are preserved.

Evidence:
- `internal/importer/importer.go` reads only `~/.claude.json` `mcpServers`.
- Redaction runs only over `server.Get("env")`.

Needed fix: either expand import or keep it explicitly scoped in user docs and
tests. If command args remain importable, redact obvious secret values there too.

### P2 — Skills-only apply can rewrite JSON configs

The docs previously implied skills-only apply was link-only. Current adapters
read and write tool JSON files even when the only planned changes are skill
symlinks. For OpenCode, that can remove all JSONC comments.

Evidence:
- Claude `Apply` writes `.claude.json` and settings before linking.
- OpenCode `Apply` writes `opencode.jsonc` before linking.

Needed fix: skip JSON writes when a changeset contains only `skill.*` changes, or
intentionally document this side effect everywhere.

### P2 — `doctor` is incomplete for OpenCode skill links

`doctor` checks content existence and the Claude skill symlink, but not the
OpenCode skill symlink.

Evidence:
- `internal/engine/status.go` checks `~/.claude/skills/<name>` only.

Needed fix: validate both tool link destinations and make output specific enough
to identify which tool is missing or wrong.

### P2 — CI and release checks are too narrow

CI runs only vet and tests. It should also prove that the binary builds, version
stamping works, formatting is clean, `go mod tidy` is clean, race tests pass, and
the CLI can smoke-run in a temp HOME.

Evidence:
- `.github/workflows/ci.yml` contains only `go vet ./...` and `go test ./...`.

Needed fix: add `go build`, stamped `--version`, `gofmt`, `go mod tidy -diff`,
`go test -race ./...`, and at least one CLI smoke using temp directories.

## Documentation State

Docs now distinguish current behavior from planned work:

- `docs/NEXT_AGENT.md` is the project handoff.
- `README.md` now states import/status/doctor/skills-only limitations.
- Living specs have concrete Purpose sections instead of archival `TBD`s.
- OpenCode comment loss is documented as whole-file comment removal.
- ADR 0004 now reflects per-adapter state saves.
- ADR 0005 now points to ADR 0008's warn-not-halt preflight change.

Remaining docs work:

- Add a core homonto user guide under `docs/guides/`.
- Keep roadmap and specs synchronized after each behavioral fix.
- Avoid using archived change artifacts as current truth unless they were merged
  into living specs, ADRs, or guides.

## Next Agent Start Sequence

1. Read `docs/NEXT_AGENT.md`.
2. Confirm the worktree state and avoid touching unrelated user changes.
3. Pick one open finding above, preferably state adoption or true `status`.
4. Add or update tests that reproduce the exact gap before changing behavior.
5. Run at least `rtk go test ./...`, `rtk go vet ./...`, and a focused CLI smoke
   for the changed behavior.
6. Update README/spec/roadmap text in the same change if behavior or user-facing
   limitations change.
