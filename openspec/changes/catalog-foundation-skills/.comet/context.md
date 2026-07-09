# Comet Design Handoff

- Change: catalog-foundation-skills
- Phase: design
- Mode: compact
- Context hash: 20bd9ce24d63cda40e74c8dbc898af26f300d50f19c3c7f70758163db16729a0

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/catalog-foundation-skills/proposal.md

- Source: openspec/changes/catalog-foundation-skills/proposal.md
- Lines: 1-35
- SHA256: 8d9c74ac97ba2ff8868b4bd858983809e748e3ae6b621aa25266a8d372017dc7

```md
## Why

Homonto's config model parses `[frameworks.X]`, `[commands.X]`, and `[subagents.X]` with `source = "builtin:<name>"`, but current adapters only project local-source skills. The first-release bundled frameworks (`onto`, `comet`, `superpowers`, `openspec`) cannot be installed today. This change adds the catalog foundation: a bundled, embedded catalog; framework metadata with dependency expansion; and builtin skill projection -- the base layer that command and subagent projection build on later.

## What Changes

- Add a bundled catalog directory (`catalog/`) at the repo root, embedded in the binary via `go:embed`.
- Add framework metadata files (`catalog/frameworks/<name>/framework.toml`) declaring name, version, dependencies, and bundled skill resources.
- Implement framework expansion: `[frameworks.X] source = "builtin:<name>"` expands to its constituent skills plus transitive dependency skills.
- Implement builtin source resolution for skills: `source = "builtin:<name>"` materializes from the embedded catalog to `.homonto/catalog/skills/<name>/` and symlinks from there.
- Extend both Claude Code and OpenCode adapters to handle builtin-source skills alongside existing local-source skills.
- Populate the catalog with all four first-release frameworks: `onto`, `comet`, `superpowers`, `openspec`.
- Add state tracking for materialized catalog resources, including version-aware re-materialization.

## Capabilities

### New Capabilities

- `builtin-catalog`: Bundled catalog structure, go:embed integration, materialization to `.homonto/catalog/`, and builtin source resolution for skills.
- `framework-expansion`: Framework metadata format, dependency expansion, transitive resolution, and atomic framework enable/disable semantics.

### Modified Capabilities

- `config-model`: Frameworks and skills now have projection behavior for builtin sources, not just validation.
- `tool-adapters`: Adapters resolve and project builtin-source skills from the materialized catalog path.

## Impact

- New `catalog/` directory tree with framework metadata and bundled skill content.
- New `internal/catalog/` Go package for catalog loading, framework expansion, and materialization.
- Modified `internal/config/config.go` for framework expansion hooks.
- Modified `internal/adapter/{claude,opencode}` for builtin source resolution.
- Modified `internal/engine/` for materialization orchestration.
- New tests for catalog parsing, framework expansion, materialization, and builtin skill projection.
- `homonto.toml` may use `[frameworks.comet]` instead of individual `[skills.comet]` entries.

```

## openspec/changes/catalog-foundation-skills/design.md

- Source: openspec/changes/catalog-foundation-skills/design.md
- Lines: 1-102
- SHA256: 818c7ef01b5934cff072da7a0da38f3524ab95781021c97149242c77d34ef359

[TRUNCATED]

```md
## Context

Homonto's config model accepts `[frameworks.X]` and `source = "builtin:<name>"` but current adapters only project local-source skills via symlink from `homonto/skills/<name>`. There is no bundled catalog, no framework metadata, and no builtin source resolution. The first-release frameworks (`onto`, `comet`, `superpowers`, `openspec`) cannot be installed through Homonto today.

Current state: `go:embed` is not used anywhere in the codebase. The adapter skill projection model is: resolve `local:<name>` to `homonto/skills/<name>`, create a symlink at the scope-appropriate tool skills directory, record state. Builtin source has no resolution path.

## Goals / Non-Goals

**Goals:**
- Embed a bundled catalog in the Go binary via `go:embed`.
- Define framework metadata (TOML) declaring name, version, dependencies, and skill resources.
- Expand `[frameworks.X]` into constituent skills with transitive dependency resolution.
- Materialize builtin skills to `.homonto/catalog/skills/<name>/` and project them via symlink.
- Populate the catalog with onto, comet, superpowers, and openspec frameworks.
- Extend both adapters to handle builtin-source skills.

**Non-Goals:**
- Command projection (separate change).
- Subagent projection (separate change).
- Model routing projection.
- Grouped plan output redesign.
- Remote fetching or registry.
- Per-resource framework-internal overrides.
- Converting existing `docs/specs/*.md` to OpenSpec specs.

## Decisions

### D1: Catalog layout and go:embed

```
catalog/
  frameworks/
    onto/framework.toml
    comet/framework.toml
    superpowers/framework.toml
    openspec/framework.toml
  skills/
    <name>/SKILL.md
    <name>/references/...
```

`go:embed all:catalog` embeds the tree at compile time. At runtime, the catalog is read from the embedded FS. On `apply`, builtin resources are materialized (extracted) to `.homonto/catalog/skills/<name>/` so symlinks can point at real directories.

**Alternative considered**: resolve from the embedded FS directly without materialization. Rejected because symlinks require real filesystem targets, and the embedded FS is read-only virtual.

### D2: Framework metadata format

```toml
# catalog/frameworks/comet/framework.toml
name = "comet"
version = "0.1.0"
description = "Comet dual-star development workflow"

[dependencies]
frameworks = ["superpowers", "openspec"]

[skills]
comet = "skills/comet"
comet-open = "skills/comet-open"
comet-design = "skills/comet-design"
comet-build = "skills/comet-build"
comet-verify = "skills/comet-verify"
comet-archive = "skills/comet-archive"
comet-hotfix = "skills/comet-hotfix"
comet-tweak = "skills/comet-tweak"
```

Each framework declares its dependencies and its resource lists by kind. Skills map resource-name to catalog-relative path.

### D3: Expansion and dependency resolution

When `[frameworks.comet] source = "builtin:comet"` is declared:
1. Load `catalog/frameworks/comet/framework.toml` from the embedded FS.
2. Add all listed skills to the effective desired skill set with `source = "builtin:<skill-name>"`.
3. Recursively expand dependencies: `superpowers` and `openspec` frameworks are also expanded, adding their skills.
4. Deduplicate: if a skill appears in multiple frameworks, it is projected once.
5. Resource name collisions between frameworks and explicit `[skills.X]` declarations are config errors.

Loose builtin skills (`[skills.brainstorming] source = "builtin:brainstorming"`) are resolved directly without framework expansion.


```

Full source: openspec/changes/catalog-foundation-skills/design.md

## openspec/changes/catalog-foundation-skills/tasks.md

- Source: openspec/changes/catalog-foundation-skills/tasks.md
- Lines: 1-49
- SHA256: 8907e0862ee40420ed62a995f8c61bda1d62cb2415f48796a0d81d267bc19d7a

```md
## 1. Catalog structure and content

- [ ] 1.1 Create `catalog/frameworks/{onto,comet,superpowers,openspec}/framework.toml` with name, version, dependencies, and skills tables
- [ ] 1.2 Copy skill content from `homonto/skills/` into `catalog/skills/<name>/` for all bundled skills referenced by frameworks
- [ ] 1.3 Add `catalog/version.txt` with the initial catalog version string
- [ ] 1.4 Verify all framework.toml files reference skills that exist under `catalog/skills/`

## 2. Catalog Go package

- [ ] 2.1 Create `internal/catalog/catalog.go` with embedded FS (`go:embed all:catalog`), framework metadata parser, and framework/skill lookup APIs
- [ ] 2.2 Add dependency graph builder with cycle detection and transitive expansion
- [ ] 2.3 Add materialization function: extract builtin skill from embedded FS to `.homonto/catalog/skills/<name>/`
- [ ] 2.4 Add catalog version read and comparison for re-materialization gating
- [ ] 2.5 Write unit tests for catalog parsing, expansion, cycle detection, and materialization

## 3. Config integration

- [ ] 3.1 Extend `config.Load` to expand `[frameworks.X]` into effective skill entries with builtin source
- [ ] 3.2 Add name collision detection between framework-expanded skills and explicit `[skills.X]` entries
- [ ] 3.3 Add `Config.ExpandedSkillEntriesForTool(tool)` that returns effective skills including framework expansion
- [ ] 3.4 Write config tests for framework expansion, dependency resolution, collision detection, and cycle rejection

## 4. Engine and materialization orchestration

- [ ] 4.1 Add materialization step in engine build/apply: before adapters run, materialize all builtin skills to `.homonto/catalog/skills/`
- [ ] 4.2 Track catalog version in state; gate re-materialization on version change
- [ ] 4.3 Pass materialized catalog root path to adapters alongside existing content root

## 5. Adapter changes

- [ ] 5.1 Extend claude adapter: resolve `builtin:<name>` skills to `.homonto/catalog/skills/<name>/` path
- [ ] 5.2 Extend opencode adapter: same builtin source resolution
- [ ] 5.3 Update linker managed-root check to accept `.homonto/catalog/skills/` as a valid managed root for pruning
- [ ] 5.4 Update doctor to check builtin skill content at materialized path
- [ ] 5.5 Write adapter tests for builtin skill projection, pruning, and conflict detection

## 6. Dogfood config update

- [ ] 6.1 Update `homonto.toml` to use `[frameworks.comet] source = "builtin:comet"` instead of individual `[skills.X]` entries for Comet/OpenSpec/Superpowers skills
- [ ] 6.2 Keep any skills not covered by frameworks as explicit local entries
- [ ] 6.3 Run `homonto apply --yes` and verify all skills materialize and link correctly
- [ ] 6.4 Run `homonto status` and `homonto doctor` and verify no drift and all links ok

## 7. Regression and docs

- [ ] 7.1 Run full regression: `go test ./... -count=1`, `go vet ./...`, `go build ./...`
- [ ] 7.2 Run stale-doc grep to ensure no doc claims builtin projection is unimplemented for skills
- [ ] 7.3 Update `docs/NEXT_AGENT.md` with catalog-foundation verification evidence
- [ ] 7.4 Commit all changes

```

## openspec/changes/catalog-foundation-skills/specs/builtin-catalog/spec.md

- Source: openspec/changes/catalog-foundation-skills/specs/builtin-catalog/spec.md
- Lines: 1-27
- SHA256: fe7d9be40fb6808c11ab656b296b70162ff5b4951216df928b4362e5fd41f62f

```md
## ADDED Requirements

### Requirement: Catalog loading from embedded filesystem

Homonto SHALL load the bundled catalog from the embedded Go filesystem at startup. The catalog loader SHALL parse all `catalog/frameworks/<name>/framework.toml` files and index frameworks by name. The loader SHALL provide framework lookup, skill content path resolution, and dependency graph traversal.

#### Scenario: Load all frameworks

- **GIVEN** a binary with four framework.toml files in the embedded catalog
- **WHEN** the catalog is loaded
- **THEN** four frameworks are indexed by name and each has its metadata parsed

### Requirement: Skill content materialization

Homonto SHALL materialize builtin skill content from the embedded catalog to `.homonto/catalog/skills/<name>/` before creating symlinks. Materialization SHALL be version-aware: the catalog version is tracked in state, and re-materialization occurs only when the version changes or the directory is missing.

#### Scenario: First materialization

- **GIVEN** no `.homonto/catalog/` directory exists
- **WHEN** apply runs with a config declaring builtin skills
- **THEN** all declared builtin skills are extracted to `.homonto/catalog/skills/<name>/`

#### Scenario: Version-gated re-materialization

- **GIVEN** `.homonto/catalog/` exists with state recording catalog version `0.1.0`
- **WHEN** apply runs with the same binary (same version)
- **THEN** materialization is skipped and existing directories are reused

```

## openspec/changes/catalog-foundation-skills/specs/config-model/spec.md

- Source: openspec/changes/catalog-foundation-skills/specs/config-model/spec.md
- Lines: 1-65
- SHA256: bf684950cfbf911e7f64f66d125c1d94445e9cc4908ca5a5bbece8617dd4f56f

```md
## ADDED Requirements

### Requirement: Bundled catalog embedded in binary

Homonto SHALL bundle a catalog directory tree at `catalog/` embedded in the Go binary via `go:embed`. The catalog SHALL contain framework metadata under `catalog/frameworks/<name>/framework.toml` and skill content under `catalog/skills/<name>/`. The embedded catalog SHALL be read-only at runtime.

#### Scenario: Catalog is available without external files

- **GIVEN** a Homonto binary built from a repo containing `catalog/`
- **WHEN** the binary runs on a machine without the source repo
- **THEN** the catalog frameworks and skills are accessible from the embedded filesystem

### Requirement: Builtin skill source resolution

A skill resource with `source = "builtin:<name>"` SHALL resolve its content from the embedded catalog at `catalog/skills/<name>/`. The content SHALL be materialized to `.homonto/catalog/skills/<name>/` on apply so that filesystem symlinks can point at a real directory.

#### Scenario: Builtin skill materializes on first apply

- **GIVEN** a config declaring `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** the user runs `homonto apply`
- **THEN** `.homonto/catalog/skills/brainstorming/` is created with content from the embedded catalog, and the skill is symlinked from there into the tool skills directories

#### Scenario: Builtin skill is idempotent on re-apply

- **GIVEN** a builtin skill already materialized and symlinked
- **WHEN** the user runs `homonto apply` again without any config change
- **THEN** no re-materialization occurs and the skill symlink is a no-op

### Requirement: Catalog version tracking

Homonto SHALL track the catalog version in `.homonto/state.json`. When the embedded catalog version differs from the recorded version, builtin resources SHALL be re-materialized on the next apply.

#### Scenario: Catalog upgrade triggers re-materialization

- **GIVEN** a builtin skill materialized under catalog version `0.1.0`
- **WHEN** a newer binary with catalog version `0.2.0` runs `homonto apply`
- **THEN** the builtin skill content in `.homonto/catalog/skills/<name>/` is refreshed and the state version is updated

### Requirement: Materialized catalog is generated state

The `.homonto/catalog/` directory SHALL be treated as generated cache. It SHALL NOT be committed to version control. The scaffolded `.gitignore` SHALL exclude `.homonto/` including the catalog cache.

#### Scenario: Gitignore covers catalog cache

- **GIVEN** a repo initialized with `homonto init`
- **WHEN** builtin skills are materialized to `.homonto/catalog/`
- **THEN** `git status` reports no untracked files under `.homonto/catalog/`

## MODIFIED Requirements

### Requirement: Local provider content root

Local provider content SHALL live under `homonto/` relative to the directory containing `homonto.toml`; generated state, cache, and the materialized builtin catalog SHALL live under `.homonto/` only. Current adapters resolve local-source skills (`source = "local:<name>"`) from `homonto/skills/<name>`. Builtin-source skills (`source = "builtin:<name>"`) resolve from the materialized `.homonto/catalog/skills/<name>/`. Local command, subagent, and framework content resolution is part of future framework/catalog projection work and MUST NOT be claimed as installed behavior yet.

#### Scenario: Local skill resolves from homonto/

- **GIVEN** a config with `[skills.my-skill] source = "local:my-skill"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `homonto/skills/my-skill/`

#### Scenario: Builtin skill resolves from materialized catalog

- **GIVEN** a config with `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `.homonto/catalog/skills/brainstorming/`

```

## openspec/changes/catalog-foundation-skills/specs/framework-expansion/spec.md

- Source: openspec/changes/catalog-foundation-skills/specs/framework-expansion/spec.md
- Lines: 1-57
- SHA256: a119373f266d3208b9b6233e359620709208dd8dcc50d61b0806ea184cd06d69

```md
## ADDED Requirements

### Requirement: Framework metadata format

Each framework in the catalog SHALL have a `framework.toml` metadata file declaring `name`, `version`, `description`, optional `[dependencies] frameworks` list, and resource lists by kind (`[skills]`, and later `[commands]`, `[subagents]`). Each resource entry SHALL map a resource name to a catalog-relative path.

#### Scenario: Parse framework metadata

- **GIVEN** a framework `catalog/frameworks/comet/framework.toml` with name, version, dependencies, and a skills table
- **WHEN** Homonto loads the framework
- **THEN** it exposes the framework name, version, dependency names, and a map of skill names to catalog paths

### Requirement: Framework expansion from builtin source

When config declares `[frameworks.<name>] source = "builtin:<framework-name>"`, Homonto SHALL expand the framework into its constituent skill resources, each with an effective `source = "builtin:<skill-name>"`. Expansion SHALL include transitive dependencies: all dependency frameworks SHALL also be expanded, and their skills added to the effective resource set.

#### Scenario: Framework expands to its skills

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` where comet declares 8 skills
- **WHEN** the config is loaded
- **THEN** the effective skill set includes all 8 comet skills as builtin-source resources

#### Scenario: Transitive dependency expansion

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` where comet depends on `superpowers` and `openspec`
- **WHEN** the config is loaded
- **THEN** the effective skill set includes comet's skills plus all skills from superpowers and openspec frameworks, deduplicated by name

### Requirement: Framework atomicity

A framework SHALL be enabled or disabled as an atomic unit. Individual framework-internal resources SHALL NOT be overridden or removed independently in this change. A loose builtin skill declared explicitly in `[skills.X]` with the same name as a framework skill SHALL produce a config error (name collision).

#### Scenario: Name collision between framework and loose skill

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` which includes skill `comet-open`, and also `[skills.comet-open] source = "builtin:comet-open"`
- **WHEN** the config is loaded
- **THEN** `config.Load` returns an error naming the collision

### Requirement: Dependency cycle detection

Framework dependency expansion SHALL detect cycles. If framework A depends on B and B depends on A (directly or transitively), `config.Load` SHALL return an error naming the cycle.

#### Scenario: Circular dependency rejected

- **GIVEN** framework A depends on B and B depends on A
- **WHEN** the config declares `[frameworks.A] source = "builtin:A"`
- **THEN** `config.Load` returns an error naming the circular dependency chain

### Requirement: First-release catalog frameworks

The catalog SHALL contain the four first-release frameworks: `onto`, `comet`, `superpowers`, and `openspec`. The `comet` framework SHALL declare dependencies on `superpowers` and `openspec`. The `onto` and `superpowers` and `openspec` frameworks SHALL have no dependencies.

#### Scenario: Comet framework declares correct dependencies

- **GIVEN** the bundled catalog
- **WHEN** the comet framework metadata is loaded
- **THEN** its dependencies list is `["superpowers", "openspec"]`

```

## openspec/changes/catalog-foundation-skills/specs/tool-adapters/spec.md

- Source: openspec/changes/catalog-foundation-skills/specs/tool-adapters/spec.md
- Lines: 1-44
- SHA256: 44f6be9fb8ea5d05075da71817d37399f297c46e276c9f0b0ec6ce65da614d47

```md
## MODIFIED Requirements

### Requirement: Owned content linked by symlink with conflict detection

Owned skills SHALL be linked (not copied) from their source into each tool's skills directory at the location chosen by the skill resource's `scope`, and pending link work SHALL be visible as plan changes: a missing link appears as a create, a link pointing at the wrong target appears as an update, and a correct link is a no-op. Local-source skills (`source = "local:<name>"`) SHALL be linked from `homonto/skills/<name>`. Builtin-source skills (`source = "builtin:<name>"`) SHALL be linked from the materialized catalog at `.homonto/catalog/skills/<name>`. `apply` SHALL create both local and builtin skill links even when they are the only pending changes, and SHALL record each applied link in state (`skill.<name>`: desired target path plus applied hash) so drift detection and pruning both see it. A skill removed from the config SHALL have its link pruned only when the existing path is a symlink pointing into homonto's managed content (either `homonto/skills/` for local or `.homonto/catalog/skills/` for builtin). If the target already exists and is not homonto's link, the adapter SHALL report a conflict and SHALL NOT clobber it -- for creation and for pruning alike.

#### Scenario: Idempotent link creation

- **WHEN** a skill symlink does not yet exist
- **THEN** plan lists a create for that link, apply creates it, and a second plan/apply reports no change for that link

#### Scenario: Skills-only config still applies

- **GIVEN** a config whose only content is owned skills declared as explicit `[skills.<name>]` resources
- **WHEN** the user runs `homonto apply` and confirms
- **THEN** the plan shows one create per missing link and apply creates every link

#### Scenario: Relative local content dir yields absolute link targets

- **GIVEN** homonto invoked from any working directory with the default `homonto/` local provider root
- **WHEN** apply creates skill links
- **THEN** every symlink target is an absolute path resolved against the config file's directory, and the link does not dangle

#### Scenario: Builtin skill links to materialized catalog

- **GIVEN** a config with `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is the absolute path to `.homonto/catalog/skills/brainstorming/`

#### Scenario: Conflict is reported, not clobbered

- **WHEN** the link target exists as a real file or points elsewhere
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: Applied link recorded in state

- **WHEN** apply creates a skill link
- **THEN** state contains a `skill.<name>` record, and `homonto status` reports drift if the link is later changed out-of-band

#### Scenario: De-declared skill pruned only when it is our link

- **GIVEN** a skill resource removed from `homonto.toml` whose target path is a real file (or a symlink pointing outside homonto's managed roots)
- **WHEN** apply processes the resulting delete
- **THEN** the path is left untouched and a conflict is reported; only a symlink into homonto's managed content (`homonto/skills/` or `.homonto/catalog/skills/`) is removed

```
