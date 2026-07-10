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

### D4: Materialization to .homonto/catalog/

On `apply`, before creating symlinks:
1. Read the embedded catalog version (from a `catalog/version.txt` or framework.toml versions).
2. If `.homonto/catalog/` does not exist or version differs, extract builtin skills to `.homonto/catalog/skills/<name>/`.
3. Skill symlinks then point at `.homonto/catalog/skills/<name>/` instead of `homonto/skills/<name>/`.

State records the catalog version for idempotency. A version match means no re-materialization.

### D5: Adapter changes

Both adapters already compute skill source paths via `localSourceName`. Add a `builtinSourcePath` path: when `source` starts with `builtin:`, resolve to `.homonto/catalog/skills/<trimmed-name>/`. The adapter creates the symlink from this path exactly as it does for local sources today.

The linker conflict detection (only relink symlinks whose target is inside the managed content root) must be updated to accept `.homonto/catalog/` as a valid managed root for builtin-source links.

## Risks / Trade-offs

- [Catalog size grows binary] -> Mitigation: catalog is markdown skills only in this change; size is bounded. Version metadata enables future selective materialization.
- [go:embed requires `catalog/` to exist at build time] -> Mitigation: catalog is tracked source, always present in the repo.
- [Materialization adds an apply step] -> Mitigation: only runs when version changes or on first apply; idempotent.
- [Framework expansion changes plan output] -> Mitigation: plan shows expanded skills with a framework-origin note; full grouped output is deferred.
- [State must track catalog vs local origin] -> Mitigation: state entries record source type and path; pruning checks origin to avoid removing wrong links.
