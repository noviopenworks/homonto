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
| `homonto agents list` | Implemented | Lists recorded installs from lockfile. |
| `homonto agents add` | Implemented | Copy/link install; lockfile + base blob recorded. |
| `homonto agents doctor` | Implemented | Per-agent health reporting. |
| `homonto agents update` / `--all` | Implemented | Three-way merge; conflict sidecar; bulk mode. |
| `homonto agents prune` | **Partial** | Deletion-error defect — see [Partial And Unsafe Behavior](#partial-and-unsafe-behavior). |
| Claude + OpenCode MCP/settings projection | Implemented | Surgical managed-key updates. |
| Reference-only secrets | Implemented | Resolved after confirm, never stored (`ADR 0002`). |
| Atomic writes + adoption + pruning + drift | Implemented | `ADR 0003`, `ADR 0004`, `ADR 0009`, `ADR 0010`. |
| Local + builtin skills/commands/subagents | Implemented | User or project scope (`ADR 0011`). |
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

These are known defects or unsafe paths that block the first public release.
Each names the exact source location and the missing invariant.

### Agent ownership: de-declared target records dropped on update

`runAgentUpdate` at `internal/cli/agents.go:585-590` initializes
`installedRec` to an empty map and writes entries only for the agent's
*currently-declared* targets. When a target is removed from the config and
`agents update` runs, the prior install record is silently dropped from the
lockfile — even though the file may still be on disk. The result is an
**untracked install**: Homonto no longer remembers it owns the file, so
`status`, `doctor`, and a later `prune` cannot reason about it correctly.

- **Invariant violated:** never forget ownership while a managed file remains
  on disk.
- **Fix scope:** carry forward prior records for de-declared targets (mark them
  for prune) instead of rebuilding `installedRec` from declared targets only.
- **Status:** open — no source fix, no focused regression test.

### Agent ownership: prune deletion failures ignored

`pruneFile` at `internal/cli/agents.go:90-91` calls `os.Remove(ti.Path)` and
`os.Remove(ti.Path+".merged")` and **discards both return values**. A failed
deletion (permission, read-only filesystem, etc.) is therefore reported as
`removed`, the lockfile record is deleted, and ownership is lost. The existing
test `TestAgentsPruneBackupFailureKeepsFile` in
`internal/cli/agents_prune_test.go` covers **backup** failure only; no test
covers **deletion** failure.

- **Invariant violated:** failed deletion cannot produce a false "removed"
  report or drop ownership.
- **Fix scope:** check both `os.Remove` errors; on failure keep the record and
  return `false` so the agent is not reported pruned.
- **Status:** open — no source fix, no focused regression test.

### Docker image builds `homonto` only

`test/docker/Dockerfile` runs `go build -o /usr/local/bin/homonto .` and the
smoke script exercises only `homonto`. There is no `onto` binary in the image
and no lifecycle suite for either binary's expanded surface (catalog
materialization, agents, onto phases, release packaging).

### Release workflow gate weaker than CI

`.github/workflows/release.yml` `verify` step runs only `gofmt -l`, `go vet`,
and `go test`. It **omits** `go test -race`, `go mod tidy -diff`, the Docker
E2E, and `govulncheck` — all of which CI (`ci.yml`) runs. A pushed tag can
therefore publish artifacts on a weaker gate than a normal pull request.
There is also no single shared gate command; local rehearsal and CI inline
overlapping but not identical checks.

## Not Implemented

These capabilities are **Deferred** or **Planned** and are not present in
source. Each is ordered as future work in the backlog or twelve-month
direction.

- **Remote sources** — remote frameworks, skills, commands, subagents, and
  agents. Deferred pending a threat model (backlog item 10).
- **Third tool adapter** — only Claude and OpenCode ship today. Deferred
  pending an adapter contract (backlog item 11).
- **Interactive Homonto TUI** — Deferred; current surface is plan/confirm/apply.
- **OpenCode comment preservation** — writes do not preserve user comments in
  OpenCode JSON; Deferred unless demand outweighs complexity.
- **Broad config import** — `homonto import` is narrow by design.
- **Blob garbage collection** — unreferenced `agentblob` content is not
  reclaimed. Planned (backlog item 9).
- **Per-agent scope semantics** — agents are not yet scoped or relocated with
  a documented model. Planned (backlog item 9).

## Current Release Gate

`v0.1.0` is **not yet cut** (`git tag --list` is empty). Four release-integrity
items must close before `v0.1.0-rc.1`. All four are **open**.

- [ ] **1. Agent ownership safety.** Exit gate: de-declared target records
  survive `agents update`; deletion failure is treated as prune failure and
  retains ownership; focused regression tests cover both invariants.
  *See backlog item 1.*
- [ ] **2. Documentation truth.** Exit gate: one status authority (this file);
  no stale capability claims in README, guides, or OpenSpec main specs; test
  counts and release status stated in one place. *See backlog item 2.*
- [ ] **3. Dual-binary Docker E2E.** Exit gate: the Docker image builds both
  `homonto` and `onto`; five suites (`homonto-core`, `homonto-expanded`,
  `homonto-agents`, `onto-lifecycle`, `release-packaging`) pass against
  disposable state. *See backlog item 4.*
- [ ] **4. Unified release gate.** Exit gate: one shared gate command drives
  local rehearsal, CI, and release publication; a tag cannot publish unless
  the complete gate (race, mod-tidy, vet, build, test, Docker E2E,
  govulncheck, packaging smoke) passes. *See backlog items 5–6.*

## Implementation Backlog

Dependency-ordered. Later work is not ready while an earlier safety or release
gate remains open without a recorded exception.

### 1. Agent Ownership Safety — *open*

- **Problem:** two ownership defects (above) can silently drop lockfile
  records while files remain on disk, breaking the never-forget-ownership
  invariant.
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

### 2. Documentation Consolidation — *in progress (this change)*

- **Problem:** project truth is scattered and partly contradictory across
  README, two release docs, `docs/specs/`, OpenSpec, guides, and historical
  plans; some sources claim safety work the source does not implement.
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

### 3. Scaffold And Contract Drift — *open*

- **Problem:** scaffolded plugin examples, `agents`/`doctor` help, and
  remediation text drift from the implemented command surface; six OpenSpec
  main specs carry generated `TBD` Purpose sections.
- **Scope:** replace obsolete plugin examples; correct help/remediation copy;
  write concrete Purpose sections; correct OpenSpec claims about builtin
  catalog support, agent effective mode, and the command surface.
- **Dependencies:** item 2 (authority established first).
- **Primary files:** scaffold templates under `catalog/`, `internal/cli`
  command help, `openspec/specs/*/spec.md`.
- **Acceptance:** scaffolds compile/run against current source; every OpenSpec
  main spec has a concrete Purpose and no stale capability claim.
- **Verify:** scaffolded `homonto init` output applies cleanly; OpenSpec
  purpose scan finds no `TBD`.
- **Exit gate:** no documented command or schema contradicts source.

### 4. Dual-Binary Docker End-to-End — *open*

- **Problem:** the Docker image and smoke cover only `homonto` core; no
  evidence for `onto`, expanded `homonto` projection, agents, or release
  packaging.
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

### 5. Release Artifact Smoke — *open*

- **Problem:** no suite verifies the published archives, checksums,
  extraction, or stamped version strings.
- **Scope:** smoke the 12 cross-compiled archives (3 OS × 2 arch × 2
  binaries), `SHA256SUMS`, archive extraction, and disposable-home
  `version`/`init`/`plan`/`apply`/`status`.
- **Dependencies:** item 4 (packaging suite reuses the dual-binary image).
- **Primary files:** `scripts/build-release.sh`, `docs/release-checklist.md`.
- **Acceptance:** a built release tree passes extraction + checksum + smoke.
- **Verify:** `scripts/build-release.sh "$(git describe)"` then archive smoke.
- **Exit gate:** release artifacts smoke-clean before any tag.

### 6. Unified Release Gate — *open*

- **Problem:** local rehearsal, CI, and release publication run different
  check sets; the release workflow's gate is strictly weaker than CI.
- **Scope:** one repository command (script/Makefile target) drives all three
  paths; release publication depends on the complete gate; prevent publishing
  while a CI job is failing or pending.
- **Dependencies:** items 4–5 (gate includes Docker E2E and packaging smoke).
- **Primary files:** `.github/workflows/release.yml`,
  `.github/workflows/ci.yml`, new shared gate script.
- **Acceptance:** the release workflow invokes the same command CI and local
  rehearsal use.
- **Verify:** tag a dry-run; confirm the gate runs end-to-end.
- **Exit gate:** a tag cannot publish unless the complete gate passes.

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

### 8. Public Stabilization — *planned*

- **Problem:** failure-path coverage is thin in selected filesystem and
  drift-observation paths; no fuzz/property tests; oversized files impede
  review.
- **Scope:** add failure-path coverage for atomic writes, link removal, state
  saves, `ObserveHashes`, and doctor helpers; add fuzz/property tests for
  config parsing, JSON-path escaping, merge invariants, and state
  serialization; split `internal/cli/agents.go` behind characterization tests.
- **Dependencies:** item 7 (stabilize after RC, not before).
- **Primary files:** `internal/cli/agents*.go`, `internal/fsutil/`,
  `internal/merge/`, `internal/agentlock/`.
- **Acceptance:** failure paths covered; fuzz seeds committed; split files
  preserve public behavior.
- **Verify:** `go test -race ./...`; `go test -fuzz` seeds pass.
- **Exit gate:** `v0.1.0` promoted only after a clean RC cycle.

### 9. Resource Coherence — *planned*

- **Problem:** `[agents]` and `[subagents]` overlap without a documented
  ownership/lifecycle model; no per-agent scope; no blob GC; conflict
  resolution is not fully recoverable.
- **Scope:** reconcile `[agents]`/`[subagents]` with a migration path; define
  per-agent scope and relocation; add target compatibility metadata; add safe
  content-addressed blob GC; make `.merged` resolution explicit and
  recoverable.
- **Dependencies:** item 8 (stable surface to reconcile against).
- **Primary files:** `internal/agentlock/`, `internal/agentblob/`,
  `internal/cli/agents.go`, `internal/config/`.
- **Acceptance:** every managed resource has one declaration model, owner,
  scope, compatibility contract, and removal path; existing configs have a
  tested migration.
- **Verify:** migration tests; GC reclaims only unreferenced blobs.
- **Exit gate:** ownership/scope model documented and tested.

### 10. Remote Trust — *planned*

- **Problem:** accepting remote resources (frameworks, skills, agents) without
  a threat model would reuse local-source assumptions against untrusted input.
- **Scope:** design one remote-source model; require immutable versions or
  content hashes in lockfiles; define provenance, cache, offline, rollback,
  revocation, and update behavior; threat-model redirects, traversal,
  symlinks, oversized content, malicious archives, compromised registries,
  and dependency substitution. Non-goal: automatic remote updates in the first
  increment.
- **Dependencies:** item 9 (coherent resource model first).
- **Primary files:** new remote-source package, `internal/agentlock/`,
  threat-model design doc.
- **Acceptance:** a remote install is pinned, auditable, reproducible,
  cacheable, and removable; malformed/compromised content fails before any
  mutation.
- **Verify:** malicious-fixture suite fails closed.
- **Exit gate:** remote trust model approved and enforced by tests.

### 11. Ecosystem Expansion — *planned*

- **Problem:** adding a tool adapter today means copying the entire
  Claude/OpenCode control flow.
- **Scope:** publish an adapter contract and real-config compatibility fixture
  format; pilot one additional tool adapter; establish catalog governance,
  versioning, deprecation, and provenance policies. Non-goal: multiple new
  adapters in parallel.
- **Dependencies:** item 10 (provenance policies reuse remote-trust work).
- **Primary files:** `internal/adapters/` (new), `catalog/`, adapter contract
  design doc.
- **Acceptance:** a third adapter ships without duplicating Claude/OpenCode
  control flow; catalog additions have automated compatibility/provenance
  checks.
- **Verify:** third-adapter fixture suite passes.
- **Exit gate:** adapter contract published; one pilot adapter green.

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

- [x] `go test ./... -count=1` → **443 passed in 26 packages** (2026-07-11).
- [x] `go vet ./...` → clean (no issues).
- [x] `go build ./...` → success (both `homonto` and `onto` build).
- [x] `./scripts/docker-test.sh` → `SMOKE PASS` (homonto core smoke).
- [x] `git tag --list` → empty (no release cut yet; `v0.1.0-rc.1` pending the
  release gate above).

Coverage and race details: `go test -race ./...` is part of the intended
release gate (item 6) and is clean on CI; the local re-baseline above uses
`-count=1` without `-race` for speed. Prefer re-running the named command over
trusting a hand-maintained total.
