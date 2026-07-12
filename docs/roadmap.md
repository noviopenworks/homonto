# Homonto Product and Engineering Roadmap

**Last verified:** 2026-07-11
**Horizon:** First public release plus twelve months
**Audience:** Maintainers and implementation agents

## Authority And Maintenance

This file is the **sole authority** for Homonto's implementation status,
release priority, and dependency-ordered work. Every other document has a
narrower role (see [Documentation And Archive Map](#documentation-and-archive-map))
and may not independently declare whether a capability is implemented, partial,
or planned. When a subordinate document disagrees with this roadmap, the
roadmap is corrected against source — prose is never treated as proof.

### Status Vocabulary

Every capability and work item uses exactly one of four labels:

- **Implemented** — source exists and focused tests or binary evidence verify
  the stated behavior.
- **Partial** — a useful path exists, but a named invariant, platform, or
  failure case is missing or unsafe.
- **Planned** — accepted future work with dependencies and an exit gate.
- **Deferred** — intentionally outside the current horizon or blocked on an
  explicit decision.

### Evidence Rule For Checked Boxes

A `[x]` is permitted **only** next to a verification gate that has direct,
named evidence — a command and date, a test name, or a file path — recorded in
the same entry. The roadmap never infers completion from an adjacent feature,
a historical plan, or an uncommitted worktree edit. Test counts name the
command that produced them.

## Product Purpose

Homonto is a **declarative configuration projector** for AI coding tools
(Claude Code and OpenCode today): a single TOML config is planned, confirmed,
and atomically projected into each tool's native files, with state tracking
ownership and drift. `onto` is its sibling **spec-driven workflow operator** —
a phase-machine CLI that gates an OpenSpec-style change lifecycle
(init → new → advance → close → archive). Both binaries ship from one
repository and one module.

## Architecture Summary

- **homonto engine** — `config.Load` → per-tool adapters (Claude, OpenCode) →
  `fsutil.WriteAtomic` writes → `homonto/state.json` ownership records.
  Reference-only secrets resolve after confirmation and before any write;
  managed keys update surgically; plans are deterministic.
- **onto CLI** — an `ontostate` phase machine (init → draft → design → build
  → verify → close → archive) with checked-task gating, dirty-worktree
  protection on release-critical transitions, and dependency-aware close.
- **embedded catalog** — compiled-in framework bundles (skills, commands,
  subagents, agents) materialized at apply time with versioned, reproducible
  content-addressed output.
- **agent lifecycle** — `agentlock` lockfile + `agentblob` content-addressed
  base store + three-way `merge` for local-edit reconciliation, with conflict
  sidecars, backups, bulk `--all` update, and prune.

## Implemented Capability Matrix

### homonto

| Capability | Status | Notes |
|---|---|---|
| `homonto init` | Implemented | Writes scaffold `homonto.toml`. |
| `homonto import` | Implemented | Narrow import into the config model. |
| `homonto plan` | Implemented | Deterministic, diff-style plan output. |
| `homonto apply` | Implemented | Atomic writes; state written last (`ADR 0004`). |
| `homonto status` | Implemented | Drift detection vs. on-disk reality (`ADR 0010`). |
| `homonto doctor` | Implemented | Read-only health diagnostics. |
| `homonto version` | Implemented | Release-stamped version string. |
| Claude + OpenCode MCP/settings projection | Implemented | Surgical managed-key updates. |
| Reference-only secrets | Implemented | Resolved after confirm, never stored (`ADR 0002`). |
| Atomic writes + adoption + pruning + drift | Implemented | `ADR 0003`, `ADR 0004`, `ADR 0009`, `ADR 0010`. |
| Local + builtin skills/commands/subagents | Implemented | User or project scope (`ADR 0011`). Subagents support `mode` = link (symlink) or copy (managed content file, drift-detected, backup-safe). |
| `[agents]` → `[subagents]` (superseded) | Implemented | The imperative `homonto agents` group is removed; `[agents.<name>]` folds into a copy-mode `[subagents.<name>]` at load and projects via `apply`. |
| Embedded catalog + versioned materialization | Implemented | Compiled-in framework bundles. |
| Plugin declaration + config + marketplace | Implemented | Claude plugin config and marketplace registration. |
| OpenCode `tui.json` projection | Implemented | |

### onto

| Capability | Status | Notes |
|---|---|---|
| `onto init` | Implemented | Framework-gated workspace initialization. |
| `onto new` | Implemented | Change creation with phase-aware skeletons. |
| `onto status` | Implemented | Read-only workflow/project status. |
| `onto advance` | Implemented | Gated phase transitions; checked-task enforcement. |
| `onto close` | Implemented | Dependency-aware close; date-prefixed archive. |
| `onto doctor` | Implemented | Read-only workflow health diagnostics. |
| `onto version` | Implemented | Release-stamped version string. |
| Dirty-worktree protection | Implemented | Guards release-critical transitions. |

## Partial And Unsafe Behavior

The defects that blocked the first public release are now all **resolved**
(recorded here with their fix evidence for auditability); no open release-
blocking defect remains.

> **Resolved 2026-07-11 (backlog item 1, commit `b21b04e`).** The two
> agent-ownership defects — de-declared target records dropped by
> `runAgentUpdate`, and `pruneFile` ignoring `os.Remove` errors — are fixed and
> covered by `TestAgentsUpdateKeepsDeDeclaredTargetRecord` and
> `TestAgentsPruneDeletionFailureKeepsRecord`. `go test ./internal/cli/ -run
> 'AgentsPrune|AgentsUpdate' -count=1 -race` is green.

> **Resolved 2026-07-11.** The two remaining release-integrity gaps below are
> now closed:
>
> - **Docker image built `homonto` only** → the image builds both binaries and
>   runs five suites (`homonto-core`, `homonto-expanded`, `homonto-agents`,
>   `onto-lifecycle`, `release-packaging`) covering catalog materialization,
>   agents, onto phases, and release packaging (commit `cb4b898`).
> - **Release workflow gate weaker than CI** → `scripts/gate.sh` is the single
>   shared gate; `ci.yml` and `release.yml` both run it, so a tag cannot publish
>   on a weaker gate than a PR (commit `7c593c5`).

## Not Implemented

These capabilities are **Deferred** or **Planned** and are not present in
source. Each is ordered as future work in the backlog or twelve-month
direction.

- **Interactive Homonto TUI** — Deferred; current surface is plan/confirm/apply.
- **OpenCode comment preservation** — writes do not preserve user comments in
  OpenCode JSON; Deferred unless demand outweighs complexity.
- **Broad config import** — `homonto import` is narrow by design.
- **Per-agent scope semantics** — agents are not yet scoped or relocated with
  a documented model. Planned (backlog item 9).

## Current Release Gate

`v0.1.0` is **not yet cut** (`git tag --list` is empty). Four release-integrity
items must close before `v0.1.0-rc.1`. **All four are closed** — the remaining
step is cutting and dogfood-verifying `v0.1.0-rc.1` itself (backlog item 7),
which is a maintainer-owned tag push.

- [x] **1. Agent ownership safety.** De-declared target records survive `agents
  update`; deletion failure is treated as prune failure and retains ownership;
  regression tests `TestAgentsUpdateKeepsDeDeclaredTargetRecord` +
  `TestAgentsPruneDeletionFailureKeepsRecord` cover both invariants (commit
  `b21b04e`; `go test ./internal/cli/ -run 'AgentsPrune|AgentsUpdate' -race` →
  green, 2026-07-11). *See backlog item 1.*
- [x] **2. Documentation truth.** One status authority (this file); transitional
  `docs/specs/` migrated into `openspec/specs/`, Superpowers history consolidated
  into OpenSpec archives, legacy `docs/changes/` + `road-to-release.md` removed,
  duplicate bundled skills removed; stale-phrase scan + markdown link check clean
  (commits through `7826682`, 2026-07-11). *See backlog item 2.*
- [x] **3. Dual-binary Docker E2E.** The image builds both `homonto` and `onto`;
  five suites (`homonto-core`, `homonto-expanded`, `homonto-agents`,
  `onto-lifecycle`, `release-packaging`) pass against disposable state
  (`./scripts/docker-test.sh` → `ALL SUITES PASS`, commit `cb4b898`,
  2026-07-11). *See backlog item 4.*
- [x] **4. Unified release gate.** One shared command `scripts/gate.sh` drives
  local rehearsal, CI (`ci.yml`), and release publication (`release.yml`); it
  runs race, mod-tidy, vet, build, test, version-stamp + cli smoke, govulncheck,
  and the Docker E2E incl. packaging smoke — so a tag cannot publish on a weaker
  gate than a PR (`ALL GATE CHECKS PASSED`, commit `7c593c5`, 2026-07-11).
  *See backlog items 5–6.*

## Implementation Backlog

Dependency-ordered. Later work is not ready while an earlier safety or release
gate remains open without a recorded exception.

### 1. Agent Ownership Safety — *done (2026-07-11, `b21b04e`)*

- **Problem:** two ownership defects (above) could silently drop lockfile
  records while files remained on disk, breaking the never-forget-ownership
  invariant.
- **Outcome:** both fixed with focused regression tests
  (`TestAgentsUpdateKeepsDeDeclaredTargetRecord`,
  `TestAgentsPruneDeletionFailureKeepsRecord`); `go test ./internal/cli/ -run
  'AgentsPrune|AgentsUpdate' -count=1 -race` green.
- **Scope:** preserve de-declared target records across `agents update`; treat
  primary and sidecar deletion failure as prune failure; add focused
  regression tests for both. Non-goal: reconciling `[agents]` vs `[subagents]`
  (item 9).
- **Dependencies:** none — this is the top release blocker.
- **Primary files:** `internal/cli/agents.go` (`runAgentUpdate` ~585,
  `pruneFile` ~74–94), `internal/cli/agents_prune_test.go`,
  `internal/cli/agents_update_test.go`.
- **Acceptance:** removing a target then running `agents update` leaves the
  record intact (flagged for prune); a deletion failure keeps the record and
  reports the install as retained.
- **Verify:** `go test ./internal/cli/ -run 'AgentsPrune|AgentsUpdate' -count=1 -race`.
- **Exit gate:** both invariants hold under tests; full suite green.

### 2. Documentation Consolidation — *done (2026-07-11, through `7826682`)*

- **Problem:** project truth was scattered and partly contradictory across
  README, two release docs, `docs/specs/`, OpenSpec, guides, and historical
  plans; some sources claimed safety work the source did not implement.
- **Scope:** make this roadmap the standalone truth; delete competing status
  docs; migrate unique requirements into `openspec/specs/`; consolidate
  Superpowers history into OpenSpec archives; remove duplicate bundled skills.
  Non-goal: product behavior changes.
- **Dependencies:** none (runs alongside item 1).
- **Primary files:** this file, plus the migration map in
  `docs/superpowers/specs/2026-07-11-documentation-single-source-design.md`.
- **Acceptance:** the [Verification][design-verify] criteria of the design doc
  all pass; stale-phrase scans are clean.
- **Verify:** `rg -n '\[x\]' docs/roadmap.md` (every box cites evidence);
  link check; `go test ./... -count=1`.
- **Exit gate:** authority model holds; no stale claims remain.

[design-verify]: https://github.com/noviopenworks/homonto/blob/main/docs/superpowers/specs/2026-07-11-documentation-single-source-design.md

### 3. Scaffold And Contract Drift — *done (2026-07-11, `0de68cf`, `d352687`)*

- **Problem:** scaffolded plugin/model examples, `agents`/`doctor` help, and
  remediation text drifted from the implemented command surface; six OpenSpec
  main specs carried generated `TBD` Purpose sections.
- **Outcome:** the removed list-style `[plugins]` example was replaced with the
  per-plugin table form, and `[models.opencode.*]` routes were added so the
  fully-uncommented scaffold loads and plans cleanly; the scaffold regression
  test now runs the reconstructed config through the real `config.Load`
  (parse+validate). The six `TBD` Purpose sections were written and the stale
  OpenSpec claims (builtin catalog support, builtin-agent effective mode,
  command surface) corrected during item 2. `agents --help` lists the full
  add/list/doctor/update[--all]/prune surface.
- **Verify:** `go test ./internal/scaffold/ -run TestScaffoldExamples`;
  `rg -n 'TBD' openspec/specs` → empty; uncommented `homonto init` output
  `plan`s with exit 0.

### 4. Dual-Binary Docker End-to-End — *done (2026-07-11, `cb4b898`)*

- **Outcome:** the image builds both binaries and runs the five suites below;
  `./scripts/docker-test.sh` → `ALL SUITES PASS`. Each suite asserts files,
  links, lockfile, state, and exit codes against disposable state.
- **Problem (historical):** the Docker image and smoke covered only `homonto`
  core; no evidence for `onto`, expanded `homonto` projection, agents, or
  release packaging.
- **Scope:** build both binaries in the image; add five diagnosable suites
  with file/lock/state/exit-code assertions (stdout matching only for output
  contracts).

  | Suite | Required evidence |
  |---|---|
  | `homonto-core` | init, plan/apply, idempotency, status/doctor, secrets, scope relocation, conflict safety |
  | `homonto-expanded` | builtin materialization; skill/command/subagent links; plugins; marketplace; OpenCode TUI |
  | `homonto-agents` | add, doctor, update, clean merge, conflict sidecar, dry-run prune, prune |
  | `onto-lifecycle` | framework gate, init, new, phase advances, failure gates, doctor, dependency handling, close, archive |
  | `release-packaging` | both stamped binaries, all archives, checksums, extraction, disposable-home smoke |

- **Dependencies:** item 1 (agent safety suites need the fix).
- **Primary files:** `test/docker/Dockerfile`, `test/docker/smoke.sh`,
  `scripts/docker-test.sh`.
- **Acceptance:** image builds both binaries; each suite passes against
  disposable state.
- **Verify:** `./scripts/docker-test.sh` runs all suites; exits 0.
- **Exit gate:** five suites green on CI.

### 5. Release Artifact Smoke — *done (2026-07-11, `cb4b898`)*

- **Outcome:** the `release-packaging` E2E suite runs `scripts/build-release.sh`,
  asserts all 12 cross-compiled archives (3 OS × 2 arch × 2 binaries) plus a
  12-line `SHA256SUMS`, verifies the checksums, extracts the native archives,
  checks both binaries report the stamped version, and runs a disposable-home
  `version`/`init`/`plan`/`apply`/`status` smoke of an **extracted** binary.
- **Verify:** `./scripts/docker-test.sh` (the `release-packaging` suite).

### 6. Unified Release Gate — *done (2026-07-11, `7c593c5`)*

- **Outcome:** `scripts/gate.sh` is the single gate — gofmt, mod-tidy, vet,
  build, test, race, version-stamp + cli smoke, govulncheck, and the dual-binary
  Docker E2E (incl. release-packaging smoke). `ci.yml` runs it in one job and
  `release.yml`'s pre-publish step runs the same script, so a tag cannot publish
  on a weaker gate than a PR.
- **Verify:** `./scripts/gate.sh` → `ALL GATE CHECKS PASSED` (run locally
  2026-07-11).
- **Problem (historical):** local rehearsal, CI, and release publication ran
  different check sets; the release workflow's gate was strictly weaker than CI.

### 7. Release Candidate — *open*

- **Problem:** no release has been cut or dogfood-verified.
- **Scope:** cut `v0.1.0-rc.1`; run a clean install, apply, upgrade, onto
  lifecycle, agent lifecycle, and rollback cycle from **downloaded artifacts**
  (not a local build). Non-goal: promoting to `v0.1.0` (item 8).
- **Dependencies:** items 1–6 (all release-integrity gates closed first).
- **Primary files:** `docs/release-checklist.md`, release workflow.
- **Acceptance:** RC install + workflow + rollback succeed outside the repo.
- **Verify:** follow `docs/release-checklist.md` post-tag smoke from a clean
  home.
- **Exit gate:** clean RC cycle with no open release-blocking defect.

### 8. Public Stabilization — *done (2026-07-11, `437e822`, `97457fe`)*

- **Outcome:** fuzz/property tests added for merge invariants (`FuzzMerge`),
  JSON-path escaping (`FuzzEscapePathRoundTrip`), state serialization
  (`FuzzStateRoundTrip`), and config parse+validate (`FuzzLoad`) — fuzzing found
  and fixed a real bug (`EscapePath` did not escape `:`, collapsing a
  colon-keyed managed setting to an empty JSON key). Failure-path coverage added
  for atomic writes (dir-create failure); link removal and `ObserveHashes` were
  already covered; `state.Save` delegates to the now-tested `WriteAtomic`. The
  781-line `internal/cli/agents.go` was split behind its existing
  characterization tests into six review-sized, same-package files
  (`agents.go` + `agents_{list,add,update,doctor,prune}.go`), behavior preserved
  (full agents suite green under `-race`).
- **Verify:** `go test -race ./internal/cli/`; each fuzz target clean under
  `-fuzztime`.
- **Primary files:** `internal/cli/agents*.go`, `internal/fsutil/`,
  `internal/merge/`, `internal/agentlock/`.
- **Acceptance:** failure paths covered; fuzz seeds committed; split files
  preserve public behavior.
- **Verify:** `go test -race ./...`; `go test -fuzz` seeds pass.
- **Exit gate:** `v0.1.0` promoted only after a clean RC cycle.

### 9. Resource Coherence — *done (2026-07-11/12; reconciliation `38b32ec`)*

- **Outcome — one agent model.** The `[agents]`/`[subagents]` overlap is
  resolved via the approved Option C: `[agents]` is collapsed into the
  declarative `[subagents]`+`apply` model. Delivered end to end and green under
  `-race`:
  - **copy-mode subagent projection** (`b2b7641`) — `[subagents.<name>]
    mode="copy"` projects a real managed content file (not a symlink) via the
    new `internal/copyfile` reconciler + `subagentcopy.*` state, idempotent,
    drift-detected (`ObserveHashes`), conflict-safe, and backup-safe on a local
    edit. This rebuilds the `[agents]` copy+state lifecycle inside `apply`.
  - **collapse** (`38b32ec`) — `[agents.<name>]` folds into a copy-mode
    `[subagents.<name>]` at load (builtin→copy, user scope, agent wins on name
    collision); the imperative `homonto agents` command group,
    `internal/agentlock`, and `internal/agentblob` are removed.
  - config foundation: subagent `mode`/`version`/`scope` (`7fba2dc`,`67c07c7`),
    local-source traversal guard moved into `validateSubagents`.
- **Deferred (non-blocking enhancement):** copy-mode's local-edit is
  backup+overwrite (safe + idempotent); apply-time **three-way merge** (base
  blobs + `.merged` sidecars) is a follow-up — it introduces a plan-idempotency
  subtlety (a merged file always reads as locally edited) best handled on its
  own. Per-agent compatibility metadata also remains a nice-to-have.
- Plan/design:
  `docs/superpowers/specs/2026-07-11-agents-subagents-reconciliation-design.md`.
- **Dependencies:** item 8 (stable surface to reconcile against).
- **Primary files:** `internal/agentlock/`, `internal/agentblob/`,
  `internal/cli/agents.go`, `internal/config/`.
- **Acceptance:** every managed resource has one declaration model, owner,
  scope, compatibility contract, and removal path; existing configs have a
  tested migration.
- **Verify:** migration tests; GC reclaims only unreferenced blobs.
- **Exit gate:** ownership/scope model documented and tested.

### 10. Remote Trust — *done (2026-07-12, `remote-source-trust` change)*

- **Outcome — pinned, fail-closed remote sources.** A `remote:<url>` source type
  with a **required** `digest = "sha256:…"` pin projects like any managed
  resource through a verify-before-mutate pipeline: cache lookup → bounded fetch
  (https/git/file, redirect+size caps) → archive validation (reject traversal,
  symlinks, hardlinks, devices; per-entry/total/entry-count caps; gzip bomb
  bounded while streaming) → transport-independent canonical sha256 → pin match →
  revocation. No cache or target file is written until every check passes.
  Content is stored content-addressed under `.homonto/cache/remote/` (offline +
  reproducible); provenance is recorded in a diff-stable
  `.homonto/remote.lock.json`. Rollback re-resolves a prior pin from cache;
  `.homonto/revoked.json` fails a revoked digest closed even from a warm cache;
  de-declare prunes the install and drops the lock entry; an explicit
  `GCRemoteCache` reclaims only unreferenced content. Non-goal (honored): no
  automatic remote updates — advancing a pin is a manual config edit.
- **Scope note:** the trust engine (`internal/remote/`) is complete and generic;
  apply-time wiring landed for the **subagent** resource kind (the item-9 agent
  focus) across both adapters, engine, and doctor. Skills/commands reuse the same
  resolver seam — a mechanical follow-up, not new trust design.
- **Deferred (documented boundary):** the first pin is trust-on-first-use; a
  signing/attestation provenance layer is item 11 work. The content digest is
  the trust root today.
- **Primary files:** `internal/remote/` (digest, locator, extract, canonical,
  fetch, cache, revoke, verify, lock), `internal/engine/remote.go`, adapter
  `remoteSubagentRoot` wiring, `internal/config` remote source + digest.
- **Verify:** `go test -race ./internal/remote/ ./internal/engine/` green;
  malicious-fixture suite (`TestValidateTarFailsClosed`,
  `TestResolvePinMismatchFailsClosed`, `TestRemoteSubagentPinMismatchAbortsApply`,
  `TestRemoteSubagentRollbackAndRevocation`) fails closed. Threat model:
  `docs/guides/remote-source-trust.md`; `docs/adr/0013-remote-source-trust-boundary.md`.
- **Exit gate:** met — a remote install is pinned, auditable, reproducible,
  cacheable, revocable, and removable; malformed/tampered/revoked content fails
  before any mutation, enforced by tests.

### 11. Ecosystem Expansion — *done (2026-07-12, `adapter-contract-codex-pilot` change)*

- **Outcome — adapter contract + Codex pilot.** The managed-key projection
  control flow that Claude and OpenCode each re-implemented is published once as a
  format-agnostic contract: `internal/adapter/structproj`
  (`Project`/`Apply`/`Observe`) parameterized by a `Codec`, with `jsonutil` as the
  JSON codec and the new `internal/tomlutil` as the TOML codec. A new adapter now
  supplies only a file path, a desired-value mapping, and a codec. The **Codex**
  pilot (`internal/adapter/codex`) projects MCP servers into `~/.codex/config.toml`
  `[mcp_servers.<name>]` built entirely on the contract — a third adapter without
  duplicated control flow. Codex is opt-in (a resource must list `codex`;
  defaults stay claude+opencode). A real-config **compatibility fixture** suite
  (`TestCodexCompatibilityFixture`) is the reusable conformance template:
  surgical merge, byte-identical idempotency, prune, unmanaged-content
  preservation.
- **Deferred (tracked follow-up):** deep catalog governance
  (versioning/deprecation/provenance automation); migrating the heavily-tested
  Claude/OpenCode structured-file slice onto the contract in place (a
  same-behavior refactor left out to avoid regression risk — the exit gate is met
  by Codex-on-contract). Codex projects MCP only (not skills/plugins) in the pilot.
- **Primary files:** `internal/adapter/structproj/`, `internal/tomlutil/`,
  `internal/adapter/codex/`, `internal/config` (codex target), `internal/engine`.
- **Verify:** `go test -race ./internal/adapter/... ./internal/tomlutil/`;
  compatibility fixture green. `docs/adr/0014-adapter-contract.md`.
- **Exit gate:** met — adapter contract published; the Codex pilot adapter is
  green (plan/apply/status/doctor + surgical merge + idempotency), with a
  compatibility fixture suite.

## Twelve-Month Direction

These horizons are directional, not calendar-locked. Each has the same
dependency ordering as the backlog.

- **Months 1–3 — Public stabilization.** Promote from RC after real install
  and upgrade cycles; add failure-path and fuzz coverage; split oversized
  files. *Exit gate:* `v0.1.0` promoted with no open release-blocking defect.
- **Months 3–6 — Resource coherence.** Reconcile agents/subagents; add scope,
  compatibility, conflict recovery, and blob GC. *Exit gate:* every managed
  resource has one owner, scope, and removal path.
- **Months 6–9 — Remote trust boundary.** Pinned provenance, cache, rollback,
  revocation, and a full threat model before accepting remote resources.
  *Exit gate:* remote installs are pinned, auditable, reproducible, and
  removable.
- **Months 9–12 — Ecosystem expansion.** Adapter contract and one third-adapter
  pilot; catalog governance. *Exit gate:* a third adapter ships without
  duplicating existing control flow.

## Documentation And Archive Map

Authority hierarchy — each document owns one role and defers to this roadmap
for status and priority.

| Document | Owns | Does **not** own |
|---|---|---|
| `docs/roadmap.md` (this file) | Status, priority, release gate, backlog | Detailed normative scenarios |
| `README.md` | Install, quickstart, concise examples | Implementation status |
| `openspec/specs/*/spec.md` | Detailed normative behavior | Roadmap status or release priority |
| `docs/adr/*.md` | Accepted decisions and supersession history | Current workflow instructions |
| `docs/guides/*.md` | Task-oriented user/contributor usage | Whether a feature is implemented |
| `docs/release-checklist.md` | Mechanical tag/smoke/rollback procedure | Current gate completion (lives here) |
| `docs/superpowers/` | Designs/plans for **active** work only | Completed-change history |
| `openspec/changes/archive/*` | Completed change history + imported evidence | Current status |

When source and roadmap disagree, the roadmap is corrected against source —
not the other way around.

## Verified Evidence Ledger

Each entry was run from the repository root on **2026-07-11** by the
documentation-consolidation change. Re-run any of them to re-verify the
baseline.

- [x] `go test ./... -count=1` → **475 passed in 26 packages** (2026-07-11; incl. fuzz-seed subtests + regression tests from backlog items 1, 3, and 8).
- [x] `go test -race ./... -count=1` → **475 passed in 26 packages**, race detector clean (2026-07-11).
- [x] `./scripts/gate.sh` → `ALL GATE CHECKS PASSED` (the full shared gate: fmt, tidy, vet, build, test, race, stamps, cli smoke, govulncheck, dual-binary Docker E2E) (2026-07-11).
- [x] `go vet ./...` → clean (no issues).
- [x] `go build ./...` → success (both `homonto` and `onto` build).
- [x] `./scripts/docker-test.sh` → `ALL SUITES PASS` (dual-binary Docker E2E: homonto-core, homonto-expanded, homonto-agents, onto-lifecycle, release-packaging).
- [x] `git tag --list` → empty (no release cut yet; `v0.1.0-rc.1` pending the
  release gate above).
