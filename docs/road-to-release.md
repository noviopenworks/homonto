# Road to Release

This is the hard release track for `homonto`. It is separate from
`docs/roadmap.md`: the roadmap is feature direction, while this file is the
gate for tagging and announcing a usable release.

## Release Verdict

Current state: **release gate reopened for the dual-binary homonto + onto
release.**

The earlier "release-ready pending the maintainer's tag" verdict is **superseded**
by the dual-binary product direction in
[`docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`](superpowers/specs/2026-07-09-dual-binary-release-design.md).
The first public release `v0.1.0-rc.1` is no longer a config-projector-only beta;
it must ship two binaries — `homonto` (deterministic installer and config
projector) and `onto` (managed spec-driven development workflow operator) —
behind the explicit per-resource config model (`[frameworks.X]`, `[skills.X]`,
`[commands.X]`, `[subagents.X]`, `[models.<tool>.<level>]` with required
`source` + `scope`, local provider content under `homonto/`).

The config-resource-model code work has landed: 168/168 tests green and
`go run . status` → `No drift` against the new model. The remaining gate is
delivering the `onto` binary, the dual-binary release packaging, and the new
coverage in the design doc's "Testing And Release Gate" section, followed by the
maintainer-owned `v0.1.0-rc.1` tag and smoke. The Iteration 0–4 history below
records the work that closed the original beta gate; it is retained as history,
not as the current release verdict.

**Onto binary foundation and `onto init` landed (2026-07-10,
`onto-binary-foundation` and `onto-init` changes, not yet merged to `main`):**
a second `package main` at `cmd/onto` now builds an `onto` binary alongside
`homonto`; `internal/ontostate` models `onto-state.yaml` (parse, validate,
derive phase; phase set `open|design|build|verify|close`); `onto status` is a
read-only, config-independent command that globs
`docs/changes/*/onto-state.yaml` and prints each active change's derived
phase, without reading `homonto.toml` or writing any file; and `onto init`
idempotently scaffolds the `docs/{changes,specs,adr,guides}` layout, gated
behind the Homonto framework install (it writes nothing if
`[frameworks.onto]` is not installed, and never overwrites user files on
repeat runs). Change skeleton creation (`onto new`, #3a) and gated phase
transitions (`onto advance`, #3b) have since landed: `onto advance <change>`
moves a change through `open → design → build → verify → close` only when
the current phase's required deliverables (and, to leave `build`, all
checked tasks) are complete, warning on a dirty worktree for a normal
advance and blocking outright on the release-critical `verify → close`
transition. Dependency resolution and archive/close rules (#3c), `onto
doctor` (#4), and dual-binary release packaging (cross-compiling and
publishing both binaries under one `SHA256SUMS`, #5) are not implemented
yet, so the dual-binary release gate above is **not** met by this work
alone.

## Iteration 0 - Safety Blockers

Goal: make the tool safe enough to recommend to users who have real Claude and
OpenCode config directories.

- [x] Fix foreign skill symlink clobbering.
  - Fixed: `link.Link`/`link.Plan` now take the content root and relink only a
    symlink whose target sits inside it; a symlink pointing outside `homonto/` is
    a user-owned conflict and is never removed or repointed.
  - Regression tests: linker level (`TestLinkForeignSymlinkIsConflict`,
    `TestPlanForeignSymlinkIsConflict`, `TestLinkRelinksManagedWrongTarget`) and
    adapter/apply level (`TestForeignSkillSymlinkAborts` in both adapters).
  - Acceptance met: an existing symlink to outside `homonto/` aborts without
    being removed or changed.

- [x] Fix or reject `settings.claude.mcpServers`.
  - Fixed: `config.Load` reserves `settings.claude.mcpServers` alongside
    `enabledPlugins` — claude's `current()` skips reading it back, so it would be
    non-idempotent.
  - Acceptance met: config load fails with a clear error; the regression case in
    `TestLoadRejectsReservedSettingKeys` names the rejected key.

- [x] Verify `status` behavior after `[skills] scope` changes.
  - Fixed: `ObserveHashes` reads each skill link at the destination state
    recorded (`recordedDst`), not the current scope's dir, so a pending scope
    switch shows as a pending relocation while old links are intact.
  - Acceptance met: `TestScopeSwitchStatusReportsPendingNotDrift` covers user ->
    project and project -> user; status reports pending relocation, no false
    drift.

- [x] Make the repository's own dogfood state clean or intentionally waived.
  - Resolved: `homonto.toml` declares each onto skill as `[skills.<name>]` with
    `scope = "project"`, so they link under this repo's own `.claude`/`.opencode`
    (gitignored) instead of the maintainer's global home. `homonto apply --yes`
    was run and verified.
  - Acceptance met: `go run . status` reports `No drift`; `doctor` shows all 8
    skills linked for both tools; the global `~/.claude` is untouched.

- [x] Sync stale docs that future agents rely on.
  - Done: current state, closed blockers, and remaining release work now live in
    this file, `docs/roadmap.md`, and the active `openspec/changes/` (the
    former `docs/NEXT_AGENT.md` handoff was retired — see ADR 0012). The guide
    index lists `using-homonto.md` as the core usage guide.
  - Acceptance met: no current-state doc contradicts source on known release
    risks.

## Iteration 1 - Release Plumbing

Goal: make a tagged build installable and auditable.

- [x] Add release CI for Linux/macOS/Windows builds.
  - `.github/workflows/release.yml` triggers on `v*` tags, re-runs the CI gates,
    and cross-compiles linux/darwin/windows for amd64+arm64 with the tag stamped
    into `homonto version`. Build+archive logic was exercised locally.
- [x] Produce checksums for release artifacts.
  - The release job writes a single `SHA256SUMS` over every `.tar.gz`/`.zip`.
- [~] Verify `go install github.com/noviopenworks/homonto@<tag>` works from a
  clean environment.
  - Installability is verified: `go install .` produces a working `homonto`
    binary from the root package. The exact `@<tag>` smoke (from outside the
    repo) is documented in the release checklist; it can only be run once a real
    tag is pushed (Iteration 4), so this stays open until the first tag.
- [x] Add `govulncheck` to CI.
  - A `govulncheck` job runs `go run golang.org/x/vuln/cmd/govulncheck@latest
    ./...`; verified locally as `No vulnerabilities found`.
- [x] Add workflow `permissions:` explicitly and keep them least-privilege.
  - `ci.yml` and `release.yml` default to `contents: read`; only the release
    job opts up to `contents: write` for publishing.
- [x] Decide whether to add CodeQL/dependency-review now or document why they
  are deferred.
  - Deferred for the v0.1.0 line, with rationale in the release checklist's
    "Security scanning decision" section (govulncheck covers the high-signal
    case for a tiny-dependency local CLI).
- [x] Add a release checklist under `docs/` covering tag, build, checksums,
  smoke install, and rollback.
  - `docs/release-checklist.md`.

## Iteration 2 - Binary-Level Coverage

Goal: reduce the gap between unit-tested internals and real CLI behavior.

- [x] Expand Docker smoke beyond skills-only apply.
  - `test/docker/smoke.sh` now covers MCP projection for Claude and OpenCode,
    settings projection, secret resolution via an env ref (asserting the value
    lands resolved in the tool files but only as a `${ref}` in state, never
    leaked), and `init` plus `import`/`import --force` command behavior.

- [x] Add a conflict smoke for real files and foreign symlinks in skill dirs.
  - The smoke pre-places a real file, then a foreign symlink, at a skill dst and
    asserts apply aborts leaving each byte-for-byte / target unchanged.
- [x] Add command-level tests for `import --force`, `init`, and error output.
  - `internal/cli/command_test.go`: init scaffolds and skips existing;
    import writes/refuses-without-force/forces; invalid and missing configs
    surface clear errors.
- [x] Stop relying only on exact human-output greps where behavior assertions can
  be made against files/state instead.
  - The new smoke sections assert against `.claude.json`, `settings.json`,
    `opencode.jsonc`, and `state.json` contents (and symlink targets), reserving
    stdout greps for genuinely output-shaped contracts (`No changes`, doctor).

## Iteration 3 - Public Beta Polish

Goal: make the first release understandable without reading internal process
docs.

- [x] Rewrite README around the user path: install, init, configure, plan,
  apply, status, doctor, limitations.
  - README leads with install → quickstart → config → secrets → scope, then a
    dedicated "Known limitations" section; internal material is quarantined.
- [x] Move or clearly quarantine internal workflow material so it does not look
  like required user documentation.
  - The "How it works" and onto "Development workflow" material now lives under a
    `## For contributors` heading ("users don't need it").
- [x] Add a short "known limitations" section to the release notes.
  - `docs/release-notes.md` carries the accepted limitations and is prepended to
    every release's auto-generated notes via the workflow's `--notes-file`.
- [x] Decide whether OpenCode JSONC comment loss remains acceptable for beta.
  - Decision: **accepted for beta**, kept loud in README, the using-homonto
    guide, and release notes. Comment-preserving writes are post-beta.
- [x] Decide whether `import` stays a narrow Claude MCP bootstrap for beta.
  - Decision: **stays narrow** for beta, documented explicitly in the README
    "Known limitations" and release notes. Expanding import is post-beta.

## Iteration 4 - v0.1.0 Release Candidate

Goal: tag only after the release surface survives a clean rehearsal.

- [x] Run full local checks — all green on 2026-07-08:
  - `gofmt -l .` clean
  - `go mod tidy -diff` clean
  - `go vet ./...` clean
  - `go build ./...` ok
  - `go test ./...` 153 passing (16 packages)
  - `go test -race ./...` 153 passing
  - `./scripts/docker-test.sh` SMOKE PASS (expanded coverage)

- [x] Run install smoke from outside the repo.
  - `go install .` produces a working `homonto` binary, and a binary copied to
    `/tmp` ran the full flow against a disposable home. The `go install
    <module>@<tag>` variant from a truly clean env is documented in the release
    checklist and can only be exercised once a tag is pushed (see below).
- [x] Run `homonto init`, edit a minimal config, `plan`, `apply --yes`,
  `status`, and `doctor` in a disposable home.
  - Rehearsed with the out-of-repo binary: init scaffolded, plan/apply projected
    a skill + setting, `status` reported `No drift`, `doctor` confirmed both
    links, and a second apply was idempotent.
- [x] Check release notes mention every accepted limitation.
  - `docs/release-notes.md` lists every accepted limitation and is prepended to
    each release's notes by the workflow.
- [ ] Tag `v0.1.0-rc.1` only after the dual-binary gate in
  [`docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`](superpowers/specs/2026-07-09-dual-binary-release-design.md)
  passes. **(Owner: maintainer.)**
  - The original beta gate (Iterations 0–4 above) is green; the reopened gate
    adds the `onto` binary, dual-binary release packaging, and the new coverage
    in the design doc. It is intentionally left to a human — it pushes a public
    tag and triggers the release workflow. Follow `docs/release-checklist.md`.
- [ ] Promote to `v0.1.0` only after at least one clean dogfood cycle with the
  tagged binary. **(Owner: maintainer.)**
  - Requires the rc tag from the previous step, so it cannot precede it.

## Non-Goals Before v0.1.0

These are useful but should not block the first release unless they become safety
issues:

- Remote framework/resource registries or marketplaces beyond the bundled
  first-release catalog.
- Per-resource overrides of framework internals.
- Plugin-specific configuration beyond current projection.
- TUI settings management.
- Agent lifecycle/version management.
- Full migration/import for every Claude/OpenCode surface.

## Current Known Commands

Last complete release rehearsal checked locally on 2026-07-08; latest
post-resource-model checks ran on 2026-07-09:

- `gofmt -l .` clean.
- `go mod tidy -diff` clean.
- `go vet ./...` clean.
- `go build ./...` passed.
- `go test ./... -count=1` passed: 168 tests in 16 packages.
- `go test -race ./...` passed in the 2026-07-08 rehearsal; rerun before tagging.
- `./scripts/docker-test.sh` passed.
- `go run . status` reports `No drift` (repo dogfooded at project scope).
- `go run . doctor` reports all 8 skills linked for both tools; only the
  environmental `pass`-not-on-PATH warning remains.
