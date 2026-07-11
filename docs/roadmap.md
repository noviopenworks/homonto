# Homonto Product and Engineering Roadmap

**Updated:** 2026-07-11
**Horizon:** First public release plus 12 months
**Audience:** Maintainers and implementation agents
**Strategy:** Release integrity first, then stabilization, model coherence,
remote trust, and ecosystem expansion.

## Purpose

This document is the forward-looking source of truth for product direction.
It records what is already implemented, what must happen next, and the gates
that prevent later work from starting too early.

Historical implementation narratives belong in archived OpenSpec changes and
release notes. Operational tag instructions remain in
[`release-checklist.md`](release-checklist.md). The executable plan for the
immediate roadmap increment is
[`2026-07-11-agentic-workflows-roadmap.md`](superpowers/plans/2026-07-11-agentic-workflows-roadmap.md).

## Product Baseline

Homonto currently ships two buildable Go binaries.

### `homonto`

Implemented commands:

- `init`, `import`, `plan`, `apply`, `status`, `doctor`, and `version`.
- `agents list`, `agents add`, `agents doctor`, `agents update`,
  `agents update --all`, and `agents prune`.

Implemented projection and lifecycle capabilities:

- Claude Code and OpenCode MCP and settings projection.
- Reference-only secrets resolved after confirmation and before writes.
- Atomic writes, surgical managed-key updates, adoption, pruning, drift
  detection, and deterministic plans.
- Local and builtin skills, commands, and subagents at user or project scope.
- Embedded framework catalog, dependency expansion, and versioned
  materialization.
- Claude and OpenCode plugin declarations; Claude plugin configuration and
  marketplace registration.
- OpenCode `tui.json` projection.
- Local and builtin lifecycle-managed agents with copy/link installation,
  lockfile state, content-addressed base blobs, three-way updates, conflict
  sidecars, backups, bulk update, health reporting, and pruning.

### `onto`

Implemented commands:

- `init`, `new`, `status`, `advance`, `close`, `doctor`, and `version`.

Implemented workflow capabilities:

- Framework-gated workspace initialization and change creation.
- Derived phases and phase-aware skeleton validation.
- Gated phase transitions, including checked-task requirements.
- Dirty-worktree protection for release-critical transitions.
- Dependency-aware close and date-prefixed archive behavior.
- Read-only workflow/project health diagnostics.

### Build and Test Baseline

- Both binaries are cross-compiled for Linux, macOS, and Windows on amd64 and
  arm64, with one checksum manifest.
- The Go suite currently contains 443 passing tests across 26 packages.
- `go test -race ./...`, `go vet ./...`, `go build ./...`, `gofmt -l .`, and
  `go mod tidy -diff` are clean as of this update.
- Statement coverage is approximately 85%, with lower coverage in selected
  filesystem failure and drift-observation paths.
- The current Docker smoke passes, but builds and exercises only `homonto` and
  primarily covers the original core projection workflow.

## Roadmap Rules

Every milestone must preserve these invariants:

- Never lose or silently overwrite user-owned content.
- Never forget ownership while a managed file remains on disk.
- Never expose resolved secrets in plans, state, logs, or test output.
- Every mutation must be idempotent or explicitly report why it cannot be.
- Living specifications and user documentation must describe current source,
  not an earlier implementation increment.
- A feature is not release-ready until it has binary-level evidence, not only
  unit tests.
- Remote input is untrusted and cannot reuse local-source assumptions without
  an explicit threat model.
- Large roadmap milestones receive a focused design and implementation plan
  before code execution begins.

## Now: Release Integrity

**Target:** Weeks 0-4
**Outcome:** A trustworthy `v0.1.0-rc.1` candidate backed by dual-binary
evidence.

### 1. Agent Lifecycle Ownership Safety

- Preserve lockfile records for de-declared targets until `agents prune`
  successfully removes their files.
- Treat install-file deletion failures as prune failures and retain ownership
  records for retry.
- Keep `.merged` cleanup errors visible without losing the primary install
  record.
- Correct stale agent command/help text and doctor remediation guidance.

Exit gate:

- Target removal followed by update cannot create an untracked install.
- Failed deletion cannot produce a false "removed" report or drop ownership.
- Focused regression tests cover both invariants.

### 2. Documentation and Specification Truth Reset

- Align README, road-to-release, release notes, and living OpenSpec
  specifications with the implemented command surface.
- Replace placeholder specification purposes and remove obsolete claims that
  builtin catalog projection or Onto lifecycle commands are missing.
- Keep completed change history out of this roadmap.
- Add a lightweight release-doc consistency checklist.

Exit gate:

- A maintainer can derive the same capability inventory from source, README,
  roadmap, release gate, and living specs.
- Current test counts and release status are stated in one place or generated.

### 3. Dual-Binary Docker End-to-End Coverage

Retain the existing Homonto core smoke and add independently diagnosable suites:

| Suite | Required evidence |
|---|---|
| `homonto-core` | Init, plan/apply, idempotency, status/doctor, secrets, scope relocation, and conflict safety |
| `homonto-expanded` | Builtin framework materialization; skill, command, and subagent links; plugins; marketplace; OpenCode TUI |
| `homonto-agents` | Local and builtin add, doctor, update, clean merge, conflict sidecar, dry-run prune, and prune |
| `onto-lifecycle` | Framework gate, init, new, phase advances, failure gates, doctor, dependency handling, close, and archive |
| `release-packaging` | Both stamped binaries, all expected archives, checksums, extraction, and disposable-home smoke |

Exit gate:

- The Docker image builds both binaries.
- Each suite runs the compiled binaries against disposable state.
- Assertions inspect files, lock/state records, exit codes, and archive layout;
  stdout matching is reserved for output contracts.

### 4. Release Gate Unification

- Provide one repository command used by local rehearsal and CI.
- Make publication depend on formatting, module tidiness, vet, build, unit tests,
  race tests, Docker E2E, vulnerability scanning, and packaging smoke.
- Prevent the release workflow from publishing while an independent CI job is
  still failing or pending.

Exit gate:

- A tag cannot publish artifacts unless the complete documented gate passes.
- The release checklist invokes the same commands as automation.

## Next: Public Stabilization

**Target:** Months 1-3
**Outcome:** Promote from release candidate after real installation and upgrade
cycles.

Priorities:

- Run at least one clean dogfood cycle using downloaded release artifacts.
- Triage RC feedback before accepting major new capability work.
- Add failure-path coverage for atomic writes, link removal, state saves,
  `ObserveHashes`, and doctor helpers.
- Add fuzz/property tests for config parsing, JSON-path escaping, merge
  invariants, and state serialization.
- Split `internal/cli/agents.go` by command responsibility without changing its
  public behavior.
- Introduce shared projection helpers only when characterization tests prove
  parity and the next resource type needs them.
- Decide whether narrow import and OpenCode comment loss remain explicit product
  limitations or receive dedicated future changes.

Exit gate:

- `v0.1.0` is promoted only after a clean RC install, apply, upgrade, Onto
  lifecycle, agent lifecycle, and rollback cycle.
- No release-blocking defect remains open without an explicit waiver.

## Then: Resource Model Coherence

**Target:** Months 3-6
**Outcome:** Remove conceptual overlap before adding remote sources.

Priorities:

- Reconcile `[agents]` and `[subagents]` into a documented ownership and
  lifecycle model with an explicit migration path.
- Define per-agent scope semantics and relocation rules.
- Add target compatibility metadata and pre-install validation.
- Add safe content-addressed blob garbage collection.
- Improve conflict resolution so resolving `.merged` state is explicit and
  recoverable.
- Expand native TUI/keybinding/theme support only for verified tool schemas and
  demonstrated user demand.

Exit gate:

- Every managed resource has one clear declaration model, owner, scope,
  compatibility contract, and removal path.
- Existing configurations have a tested migration or remain supported through a
  documented compatibility period.

## Later: Remote Trust Boundary

**Target:** Months 6-9
**Outcome:** Reproducible remote resources without weakening local safety.

Priorities:

- Design one remote-source model for frameworks, skills, commands, subagents,
  and agents where their trust requirements genuinely align.
- Require immutable versions or content hashes in lockfiles.
- Define provenance, cache, offline, rollback, revocation, and update behavior.
- Threat-model redirects, traversal, symlinks, oversized content, malicious
  archives, compromised registries, and dependency substitution.
- Add stronger static and security analysis before accepting untrusted input.
- Keep automatic remote updates out of the first remote increment.

Exit gate:

- A remote install is pinned, auditable, reproducible, cacheable, and removable.
- Compromised or malformed content fails before mutating tool configuration.

## Horizon: Ecosystem Expansion

**Target:** Months 9-12
**Outcome:** Make extension safer than copying an existing adapter.

Priorities:

- Publish an adapter contract and real-config compatibility fixture format.
- Pilot one additional tool adapter before opening broad adapter contributions.
- Establish bundled catalog governance, versioning, deprecation, and provenance
  policies.
- Expand import only for stable, fixture-backed target schemas.
- Publish contributor guides for adapters, framework bundles, and catalog
  resources.

Exit gate:

- A third adapter can be implemented without duplicating the entire
  Claude/OpenCode control flow.
- Catalog additions have automated compatibility and provenance checks.

## Explicit Deferrals

These items do not block the first stable release:

- Interactive Homonto TUI.
- Ratings, search, or a community marketplace.
- Multiple new adapters in parallel.
- Unpinned remote sources or automatic remote updates.
- Per-resource framework-internal overrides.
- Broad adapter rewrites without characterization tests.
- Comment-preserving OpenCode writes unless demand outweighs complexity.

## Agent Execution Protocol

Implementation agents must work milestone by milestone:

1. Read this roadmap, the relevant living specs, ADRs, and the milestone's
   focused implementation plan.
2. Confirm dependencies and acceptance gates before editing.
3. Use TDD for behavior changes and preserve the failing-test evidence.
4. Keep one task independently reviewable and verifiable.
5. Run the narrow test first, then the complete milestone gate.
6. Update living specs and user documentation in the same change as behavior.
7. Record verification commands and unresolved gaps before handoff.
8. Do not begin a later milestone while an earlier exit gate is unmet unless a
   maintainer records an explicit exception.

## Roadmap Health Metrics

Review these monthly and at every release:

- Release gate duration and failure causes.
- Docker E2E suite duration and flake rate by suite.
- Number of source/documentation contradictions.
- Open lifecycle ownership or data-loss defects.
- Coverage of filesystem failure paths and drift observation.
- Time from catalog/tool schema change to fixture update.
- Number of roadmap items without an owner, dependency, or exit gate.
