# Documentation Single Source of Truth Design

**Date:** 2026-07-11
**Status:** Approved design; implementation not started
**Scope:** Documentation authority, historical consolidation, tracked duplicate
cleanup, and a standalone `docs/roadmap.md`

## Context

Homonto currently distributes project truth across README content, two release
documents, a transitional `docs/specs/` tree, OpenSpec main specs, user guides,
ADRs, legacy Onto changes, OpenSpec archives, and completed Superpowers designs,
plans, and reports. Several of these sources contradict current source code or
each other.

The duplication creates concrete operational risk:

- Agents can follow obsolete plugin syntax or a retired development workflow.
- Implemented Onto and catalog behavior is still described as future work.
- Release documentation promises stronger automation than the release workflow
  executes.
- Historical plans with unchecked tasks appear active.
- The roadmap can mark safety work complete without matching source or focused
  tests.
- `catalog/skills/` and `homonto/skills/` contain duplicate tracked bundled
  content with different intended semantics.

The approved direction is a strong cleanup. `docs/roadmap.md` becomes a fully
standalone project dashboard. OpenSpec archives and accepted ADRs remain as the
only retained historical layers. Other documents may provide task-specific
depth, but they cannot independently define implementation status or priority.

## Goals

- Make `docs/roadmap.md` sufficient to understand the product, current
  implementation, defects, missing capabilities, release gate, and future work.
- Give every remaining document one narrow role and explicit authority.
- Remove stale, duplicate, superseded, and misleading tracked documents.
- Preserve useful completed design and verification evidence inside the
  corresponding OpenSpec archive before deleting scattered copies.
- Make status claims evidence-based and prevent unchecked plans from appearing
  active after archival.
- Remove duplicate bundled skill content while preserving the distinction
  between embedded catalog content and user-authored local provider content.
- Keep generated local indexes, caches, projections, and scratch data outside
  repository truth.

## Non-Goals

- No product behavior changes in the documentation-consolidation change.
- No agent lifecycle bug fixes, Docker E2E implementation, or release workflow
  implementation; the roadmap will order those as subsequent work.
- No Git history rewrite or binary archive of Markdown history.
- No deletion of accepted ADRs or OpenSpec archived changes.
- No change to the public CLI solely to accommodate documentation layout.
- No recreation of `docs/NEXT_AGENT.md` or the retired legacy handoff model.

## Authority Model

### Standalone Project Truth

`docs/roadmap.md` is the only status and priority authority. It must answer:

1. What Homonto and Onto are.
2. How the major components interact.
3. What is implemented and verified.
4. What is partial, unsafe, or known defective.
5. What is not implemented.
6. What blocks the current release.
7. What must be implemented next and in what dependency order.
8. What is explicitly deferred or rejected.
9. Where detailed requirements, decisions, guides, and historical evidence live.
10. Which commands most recently verified the stated baseline.

An engineer must not need `road-to-release.md`, old plans, archived changes, or
historical designs to determine current status.

### Subordinate Documents

- `README.md` owns installation, quickstart, concise examples, and links to the
  standalone roadmap and detailed guides.
- `openspec/specs/*/spec.md` owns detailed normative behavior. Main specs do not
  own roadmap status, release priority, or implementation history.
- `docs/adr/*.md` owns accepted architectural decisions and supersession history.
- `docs/guides/*.md` owns task-oriented user and contributor instructions. A
  guide explains how to use current behavior; it does not declare whether a
  feature is implemented.
- `docs/release-checklist.md` owns mechanical pre-tag, tag, smoke, and rollback
  procedure. Current gate completion remains in the roadmap.
- Active Superpowers designs and plans own implementation detail only while
  attached to active work.
- `openspec/changes/archive/*` owns completed change history and imported
  Superpowers evidence.
- Source and tests provide implementation evidence. When source and roadmap
  disagree, the roadmap must be corrected rather than treating prose as proof.

## Canonical Roadmap Schema

The standalone roadmap uses fixed top-level sections:

1. `Authority And Maintenance`
2. `Product Purpose`
3. `Architecture Summary`
4. `Implemented Capability Matrix`
5. `Partial And Unsafe Behavior`
6. `Not Implemented`
7. `Current Release Gate`
8. `Implementation Backlog`
9. `Twelve-Month Direction`
10. `Documentation And Archive Map`
11. `Verified Evidence Ledger`

### Status Vocabulary

Every capability uses one of four labels:

- **Implemented:** source exists and focused tests or binary evidence verify the
  stated behavior.
- **Partial:** a useful path exists, but a named invariant, platform, or failure
  case is missing or unsafe.
- **Planned:** accepted future work with dependencies and an exit gate.
- **Deferred:** intentionally outside the current horizon or blocked on an
  explicit decision.

Checked boxes are allowed only for verification gates that have direct evidence.
The roadmap must never infer completion from an adjacent feature or a historical
plan. Test counts, coverage, and dates must name the command and verification
date. Stable claims should prefer commands over manually maintained totals.

### Implementation Backlog Entries

Each work package records:

- Problem and user risk.
- Exact scope and explicit non-goals.
- Dependencies.
- Primary source/test/doc files.
- Acceptance scenarios.
- Verification commands.
- Exit gate.
- Link to an active detailed design or plan, when one exists.

The backlog is dependency ordered. Later work cannot be presented as ready while
an earlier safety or release gate remains unmet without a written exception.

## Cleanup And Migration Map

### Merge And Delete

1. Merge the current release verdict and incomplete gate from
   `docs/road-to-release.md` into the roadmap, then delete
   `docs/road-to-release.md`.
2. Migrate unique normative requirements from `docs/specs/` into matching or new
   `openspec/specs/` capabilities. Delete the entire transitional `docs/specs/`
   tree after requirement-by-requirement comparison.
3. Delete `docs/reviews/2026-07-04-deep-review.md`; current defects belong in the
   roadmap and requirements belong in OpenSpec.
4. Delete the legacy `docs/changes/archive/` tree after confirming any unique
   accepted decision is represented by an ADR or current OpenSpec requirement.
   Git history remains the recovery path for the retired Onto workspaces.
5. Merge the useful point-by-point content from
   `docs/superpowers/plans/2026-07-11-agentic-workflows-roadmap.md` into roadmap
   work packages, then delete that standalone plan.
6. Remove duplicate bundled skill files under `homonto/skills/` after a recursive
   equality check against `catalog/skills/` and source/build verification.
   Preserve `homonto/skills/.gitkeep` only if the local-provider scaffold needs an
   empty tracked directory.

### Rewrite And Reduce

- Rewrite `docs/guides/using-homonto.md` around current per-resource and plugin
  schemas, required model routes, both binaries, and current limitations.
- Rewrite `docs/guides/onto-workflow.md` exclusively as a product guide for the
  implemented Onto binary. Remove claims that the legacy skill workflow is the
  repository's current development process.
- Update `docs/guides/README.md`, `docs/adr/README.md`, and
  `docs/changes/README.md` to remove retired Onto development procedures. Delete
  `docs/changes/README.md` if the legacy directory itself is removed.
- Replace generated `TBD` Purpose sections in six OpenSpec main specs.
- Correct OpenSpec claims about builtin catalog support, agent effective mode,
  and the current command surface.
- Correct scaffolded plugin examples and CLI help/remediation in their later
  behavior changes; the roadmap must list those as implementation work until
  source changes and tests pass.

### Consolidate Superpowers History Into OpenSpec

For each completed change with a Superpowers design, plan, or verification
report:

1. Resolve the corresponding `openspec/changes/archive/<date>-<change>/`
   directory from `archived-with:` metadata or `.comet.yaml`.
2. Move the technical design to `technical-design.md` in that archive.
3. Move the implementation plan to `implementation-plan.md` in that archive.
4. Move the verification report to `verification-report.md` in that archive.
5. Update `.comet.yaml` and relative links to the new archived locations.
6. Verify no remaining tracked file references the old path.
7. Remove the completed artifact from live `docs/superpowers/` directories.

Superseded umbrella designs without a one-to-one archived change are either
attached to the final implementing archive with `superseded-by` metadata or
deleted after their unique decisions are represented by ADRs/current specs.

After migration, `docs/superpowers/` contains only:

- A README defining active-only semantics.
- Designs and plans linked to active work.
- Temporary reports for work not yet archived.

Archival must move those artifacts into OpenSpec history. Completed unchecked
plans must not remain in the active tree.

## Retained History

### OpenSpec Archives

Retain each archived change's proposal, design, tasks, delta specs,
`.openspec.yaml`, `.comet.yaml`, imported technical design, implementation plan,
and verification report.

Existing generated `.comet/` recovery material is retained during this cleanup
unless a separate archive-format design proves it is safe to thin. It is
historical process evidence, never current status.

### ADRs

Retain all ADRs, including explicitly superseded ADRs. Superseded ADR headers
must link to the replacing decision. The ADR index must describe ADRs as decision
history, not workflow instructions.

## Generated And Local Files

These paths are never authoritative and remain ignored:

- `graphify-out/`
- `.codegraph/` generated databases and daemon state
- `.homonto/`
- `.claude/` and `.opencode/` project projections
- `.superpowers/sdd/`

The implementation may remove obsolete `.superpowers/sdd/` and Graphify output
locally after checking that no current process depends on them. It may remove
CodeGraph databases only when reindexing is acceptable. It must not casually
delete `.homonto/state.json`, because that local operational state records
ownership even though it is not repository truth.

Add a root ignore rule for `/.superpowers/` so scratch policy is visible without
reading a nested ignore file.

## Current Implementation Backlog

The standalone roadmap must order these work packages:

1. **Agent ownership safety:** preserve de-declared target records across update;
   retain ownership and report failure when primary or sidecar deletion fails.
2. **Documentation consolidation:** perform this migration, correct false status
   claims, remove duplicate docs/content, and establish authority checks.
3. **Scaffold and contract drift:** replace obsolete plugin examples, correct
   agents help/remediation, and normalize OpenSpec purposes/contracts.
4. **Dual-binary Docker E2E:** retain Homonto core coverage and add expanded
   projection, agent lifecycle, and complete Onto lifecycle suites.
5. **Release artifact smoke:** verify all separate archives, checksums,
   extraction, and both stamped binaries.
6. **Unified release gate:** make local rehearsal, CI, and release publication
   execute the same complete gate.
7. **Release candidate:** cut and dogfood `v0.1.0-rc.1`, then promote only after
   a clean installation, upgrade, workflow, and rollback cycle.
8. **Public stabilization:** increase filesystem failure/drift coverage, add
   fuzz/property tests, and split oversized files behind characterization tests.
9. **Resource coherence:** reconcile agents/subagents and add scope,
   compatibility, conflict resolution, and blob garbage collection.
10. **Remote trust:** define pinned provenance, cache, rollback, revocation, and
    threat controls before accepting remote resources.
11. **Ecosystem expansion:** define an adapter contract and fixture model before
    piloting one additional tool.

The roadmap must distinguish existing implementation from acceptance evidence.
For example, normal agent prune exists, but deletion-error handling remains
partial; the current Homonto Docker smoke passes, but dual-binary and expanded
suite evidence does not exist.

## Migration Safety

The consolidation is performed in reviewable stages:

1. Establish and verify the new roadmap content before deleting competing status
   files.
2. Migrate unique requirements before deleting transitional specs.
3. Move completed Superpowers artifacts and update all archive references before
   deleting old paths.
4. Compare duplicate skills byte-for-byte immediately before removal.
5. Run link checking and stale-phrase scans after every deletion batch.
6. Run the full Go tests and existing Docker smoke because docs include embedded
   examples and catalog content removal affects builds.
7. Commit logical migration stages separately so Git history remains a clear
   recovery mechanism.

The implementation must account for an existing uncommitted modification to
`docs/roadmap.md` observed during design. It must inspect and attribute that
change before editing; it must not silently discard concurrent work. Status
claims in that delta are not accepted as evidence without source and focused
test verification.

## Verification

The cleanup is complete only when all conditions pass:

- `docs/roadmap.md` contains every fixed schema section and has no unresolved
  placeholder or unverified completion claim.
- `docs/road-to-release.md`, `docs/specs/`, the deep review, legacy Onto archives,
  and completed live Superpowers artifacts are absent.
- Every retained guide, README, ADR index, and release checklist links to the
  roadmap for project status.
- Every OpenSpec main spec has a concrete Purpose and no known stale capability
  claim.
- Every retained OpenSpec archive reference resolves after artifact migration.
- No current document references `docs/NEXT_AGENT.md` or instructs contributors
  to start work under legacy `docs/changes/`.
- `catalog/skills/` is the sole tracked bundled-skill source and the project
  still builds with embedded catalog content.
- Stale-phrase searches find no current claims that Onto, builtin projection,
  plugin config, TUI projection, or agent lifecycle are wholly unimplemented.
- Markdown links resolve.
- `git diff --check`, `go test ./... -count=1`, `go vet ./...`, and
  `go build ./...` pass.
- The existing Docker smoke passes after duplicate content removal.
- The worktree contains no generated caches or projections staged for commit.

## Risks And Mitigations

- **Lost unique historical decision:** compare each deleted legacy document
  against ADRs and current OpenSpec requirements; create an ADR only for a still
  relevant decision not represented elsewhere.
- **Broken archive references:** move artifacts and update `.comet.yaml` plus all
  Markdown links in one commit, then run a repository link scan.
- **Roadmap becoming unmaintainable:** enforce the fixed schema, short capability
  rows, evidence links, and no narrative implementation diaries.
- **Standalone roadmap duplicating detailed specs:** summarize requirements and
  link to OpenSpec for scenarios; the roadmap owns status, not every normative
  edge case.
- **Catalog/local-provider semantic collapse:** delete duplicate files but keep
  the directory semantics distinct; never symlink `homonto/skills/` to
  `catalog/skills/`.
- **Concurrent roadmap edits lost:** attribute and merge the observed worktree
  delta before implementation begins.

## Acceptance Scenarios

### New maintainer finds current truth

Given a maintainer opens the repository with no prior context, when they read
README and follow its status link, then `docs/roadmap.md` alone identifies the
implemented product, known defects, release blockers, and next work without
requiring historical documents.

### Agent does not follow stale history

Given an implementation agent searches for current work, when it encounters an
OpenSpec archive or ADR, then explicit authority text identifies it as history or
rationale, while only the roadmap declares status and priority.

### Requirements remain detailed

Given a roadmap work package references a capability, when an engineer needs
edge cases and scenarios, then the linked OpenSpec main spec provides them and
contains no conflicting implementation-status narrative.

### Historical evidence remains traceable

Given a completed OpenSpec archive, when a maintainer inspects it, then its
technical design, implementation plan, verification report, tasks, and delta
specs resolve inside the archive without links to deleted Superpowers paths.

### Duplicate content removal is safe

Given `homonto/skills/` duplicates the embedded catalog, when the duplicate files
are removed, then `catalog/skills/` remains embedded, `go build ./...` passes,
and the Homonto Docker smoke still materializes and links builtin resources.
