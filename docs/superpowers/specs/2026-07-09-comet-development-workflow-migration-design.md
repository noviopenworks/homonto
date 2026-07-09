# Comet Development Workflow Migration Design

Date: 2026-07-09
Status: Approved direction; implementation not started

## Summary

Homonto development will migrate from the repo-local `onto` markdown workflow to
Comet. Comet becomes the internal development workflow for new work in this repo:
OpenSpec owns WHAT, Superpowers owns HOW, and Comet state/scripts bind the two.

This migration is about how Homonto is developed. It does not, by itself, decide
whether `onto` remains a bundled product framework for users. Product framework
decisions stay governed by the dual-binary release design until explicitly
changed.

## Current State

The repo currently dogfoods `onto` as local skill content:

- `homonto.toml` declares `onto`, `onto-open`, `onto-design`, `onto-build`,
  `onto-verify`, `onto-close`, `onto-fix`, and `onto-tweak` as project-scoped
  local skills.
- Workflow artifacts live under `docs/changes/` and archived workspaces under
  `docs/changes/archive/`.
- Living workflow docs point agents to `/onto` and `docs/changes/state.yaml`.
- There is no active change workspace under `docs/changes/`.
- There is no initialized `openspec/` tree; `openspec list --json` currently
  fails with `No OpenSpec changes directory found`.
- There is no local Comet skill content under `homonto/skills/` today.

## Goals

- Make Comet the default development workflow for this repository.
- Initialize the OpenSpec + Comet artifact structure needed for `/comet`.
- Preserve historical `onto` archives without rewriting them.
- Keep existing Superpowers design docs and plans usable in the new workflow.
- Update living docs so future agents start with Comet, not Onto.
- Bootstrap Comet with local/project-scoped resources until framework/catalog
  projection can install `[frameworks.comet]` directly.

## Non-Goals

- No product decision to delete the user-facing `onto` concept.
- No immediate implementation of framework/catalog projection.
- No conversion of archived `docs/changes/archive/*` workspaces.
- No remote fetching, registry, marketplace, or third-party package support.
- No change to Homonto's secret/state/apply semantics.

## Recommended Architecture

Use a two-layer migration.

### Layer 1: Immediate Dogfood Bootstrap

Comet is projected like today's Onto dogfood: as local project-scoped skill
resources declared in `homonto.toml`. The repo vendors or copies the required
Comet, OpenSpec, and Superpowers skill entrypoints into `homonto/skills/` so
`homonto apply` can link them into Claude Code and OpenCode.

This layer uses only implemented Homonto behavior: local-source skill projection.
It avoids pretending `[frameworks.comet]` installs anything before catalog
projection exists.

### Layer 2: Product Catalog End State

After framework/catalog projection lands, Comet should be enabled through a
framework declaration such as:

```toml
[frameworks.comet]
source = "builtin:comet"
scope = "project"
```

At that point Homonto can replace the local skill declarations with the bundled
framework projection path and verify that dependencies (`openspec` and
`superpowers`) expand automatically.

## Artifact Layout

Comet introduces these current-development surfaces:

```text
.comet/
  config.yaml
openspec/
  config.yaml
  specs/
  changes/
docs/superpowers/
  specs/
  plans/
  reports/
```

`docs/superpowers/specs/` and `docs/superpowers/plans/` remain the HOW surfaces.
OpenSpec becomes the canonical WHAT surface for new requirement changes.

The existing `docs/changes/` tree becomes legacy Onto history. Its archive
contents remain useful historical evidence but are not the active workflow.

## Comet Defaults

Project defaults should start conservative:

```yaml
language: en
context_compression: off
auto_transition: true
```

Rationale:

- English matches current project docs.
- `context_compression: off` avoids introducing a second migration variable.
- `auto_transition: true` keeps Comet's intended phase handoff behavior while
  preserving required blocking user decision points.

## Living Documentation Changes

The migration must update living docs, not archived history:

- `README.md`: contributor workflow starts with `/comet`.
- `docs/NEXT_AGENT.md`: future agents inspect `openspec/changes/` and `.comet`
  state first.
- `docs/guides/README.md`: link to a Comet workflow guide.
- `docs/guides/comet-workflow.md`: explain local repo workflow, phases, gates,
  and artifact layout.
- `docs/specs/comet-workflow.md`: living spec for the repo's Comet workflow.
- `docs/changes/README.md`: mark as legacy Onto archive/workspace contract.
- `docs/guides/onto-workflow.md` and `docs/specs/onto-workflow.md`: either mark
  legacy/internal history or explicitly scope to the product framework if Onto
  remains product work.

## Requirements Canonicality

For new work after migration, OpenSpec specs under `openspec/specs/` are the
canonical WHAT requirements. Existing `docs/specs/*.md` files should not be
silently deleted. The migration should first add a clear bridge:

- keep `docs/specs/` readable as current documentation during the transition;
- state that new Comet-managed changes create or modify OpenSpec specs;
- plan any bulk conversion of existing `docs/specs/*.md` as a separate change if
  it becomes necessary.

This avoids mixing workflow bootstrap with a broad spec-format migration.

## Migration Boundary

The bootstrap change should do only enough to make `/comet` usable for future
development:

- initialize `.comet/` and `openspec/`;
- add/project local Comet skills;
- update living workflow docs;
- leave product release semantics explicit and unchanged unless the user makes a
  separate product decision.

The next substantive product change after bootstrap should be managed by Comet.
The likely first Comet-managed product change is framework/catalog projection for
`[frameworks.X]`, `[commands.X]`, and `[subagents.X]`.

## Risks And Mitigations

- **Risk: Comet is unavailable in OpenCode project scope.** Mitigation: vendor
  local skill resources first; do not depend on unimplemented framework
  projection.
- **Risk: Docs claim Comet is active before `openspec/` works.** Mitigation: make
  `openspec list --json` passing a migration acceptance check.
- **Risk: Onto product direction is accidentally deleted.** Mitigation: separate
  internal workflow docs from release/product docs; require a separate explicit
  decision to remove Onto from product scope.
- **Risk: Two workflow systems confuse agents.** Mitigation: mark `docs/changes/`
  legacy and update `NEXT_AGENT.md` as the authoritative handoff.
- **Risk: Bulk spec conversion balloons scope.** Mitigation: defer converting
  existing `docs/specs/*.md` until Comet is bootstrapped and working.

## Acceptance Criteria

- `openspec list --json` succeeds in a clean checkout.
- `.comet/config.yaml` exists with explicit defaults.
- `homonto.toml` projects Comet-related project-scoped local skills, or the docs
  clearly state why a manual/project-skill installation remains temporary.
- `homonto apply --yes`, `homonto status`, and `homonto doctor` pass with only
  accepted environmental warnings.
- Living docs direct future development to `/comet`, not `/onto`.
- `docs/changes/README.md` is marked legacy.
- No archived Onto workspace is rewritten.
- A first Comet change can be created under `openspec/changes/<name>/` with a
  `.comet.yaml` state file.
