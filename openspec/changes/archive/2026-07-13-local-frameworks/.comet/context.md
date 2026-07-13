# Comet Design Handoff

- Change: local-frameworks
- Phase: design
- Mode: compact
- Context hash: 8e08c004ad353df279b8c33c5d1f1382dd7dd3d142f57e615bbaa87dd44f4370

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/local-frameworks/proposal.md

- Source: openspec/changes/local-frameworks/proposal.md
- Lines: 1-45
- SHA256: bd71d55183dc1af2b51e487c27f45c5b06e48354ecd0ffe03c06675fe6ccab44

```md
# Local framework resolution end-to-end

## Why

Roadmap E1 (F36), the flagship local/custom-framework capability. The catalog
can already merge overlay framework sources (`catalog.LoadOverlays`, the
foundation) but nothing consumes it: config still rejects every non-builtin
`[frameworks.X]` (F35), the catalog materializes only from the embedded FS, and
expansion only handles `builtin:` sources. This change wires local frameworks
end-to-end so a user can install a framework from their own filesystem through
the same validated, versioned path as a builtin (E1 exit gate). Decision (D1):
`local:<path>` is structurally validated, no digest — the user owns their own
filesystem, exactly like a `local:` skill source.

## What Changes

- **Config**: accept `[frameworks.X] source = "local:<path>"`; `<path>` is a
  framework root (a `framework.toml` whose `name` equals `X`, plus its
  `skills/`/`commands/`/`subagents/` with framework-root-relative paths). Other
  non-builtin sources still fail loudly (F35 preserved for non-local).
- **Catalog**: a resource index that tracks each resource's source filesystem so
  `Materialize`/`MaterializeCommands`/`MaterializeSubagents` resolve content from
  the framework's own FS (the embedded base for builtins, the local dir for local
  frameworks). A local framework is merged from its dir via the overlay path.
- **Config expansion**: a `local:` framework's transitively-expanded resources
  project as `builtin:<name>` (they materialize into the same catalog root),
  reusing the entire existing projection path unchanged.
- **Engine**: build the catalog with the config's local-framework overlays, so
  materialization writes their content into the catalog root like a builtin.

## Impact

- **Specs:** `framework-expansion` gains a requirement that a `local:` framework
  installs through the same validated path as a builtin.
- **Behavior:** builtin-only configs are unchanged (the base FS is every
  resource's source; expansion/materialization identical). New: a `local:`
  framework's resources install.
- **Risk:** medium — new cross-subsystem behavior (config + catalog + engine).
  Guarded by an end-to-end acceptance test (a `local:` framework's skill is
  materialized by apply) plus the full existing suite (builtin path unchanged).

## Non-goals

- Remote/digest-pinned frameworks (a later phase via the trust pipeline).
- `[compat].homonto`, capabilities (later/decision-gated phases).

```

## openspec/changes/local-frameworks/design.md

- Source: openspec/changes/local-frameworks/design.md
- Lines: 1-68
- SHA256: 09ab3a656491a599b560e535146d67233b93df1f16261b1a861ab7dae5d13ce6

```md
# Design — local framework resolution end-to-end

## Model

- `local:<path>` framework: `<path>` is the framework ROOT — `framework.toml`
  (whose `name` must equal the config key) + `skills/`/`commands/`/`subagents/`
  with framework-root-relative resource paths. `<path>` is resolved relative to
  the homonto.toml directory (like other local sources).
- Merged into the catalog as an overlay framework; its resources are indexed with
  their source FS = `os.DirFS(<abs path>)`.

## Catalog: FS-aware resource index

Change the three resource indexes from `map[string]string` (name→path) to carry
the source FS too — add parallel `skillFS/commandFS/subagentFS map[string]fs.FS`
populated in `mergeSource`/the loose loops (= the source being merged; the base
for builtins). `Materialize` uses `fs.Sub(c.skillFS[name], path)` instead of
`c.fsys`; `MaterializeCommands`/`Subagents` and `SubagentContent` likewise.
Backward-compatible: for a base-only catalog every `*FS[name]` is the base, so
behavior is identical (pinned by the catalog + engine suites).

`mergeSource` today reads `frameworks/<name>/framework.toml`. A local framework
dir is a SINGLE framework at its root, so add `mergeFrameworkRoot(name string,
src fs.FS)` that reads `<src>/framework.toml`, validates `name`==the given name,
manifest schema, resource-path existence (paths are framework-root-relative), and
indexes with srcFS=src. `LoadWithLocal(base, locals map[string]fs.FS)` = base via
mergeSource + each local via mergeFrameworkRoot, then validateDependencyRanges.

## Config

- `validate` (config.go:577): allow `strings.HasPrefix(src, "local:")` for
  frameworks; keep rejecting any other non-`builtin:` source.
- A `(c *Config) frameworkCatalog(baseDir string) (*cat.Catalog, error)` builds
  the catalog with the config's `local:` frameworks (`os.DirFS(resolve(baseDir,
  path))`), replacing the global `loadedCatalog()` singleton in the three
  `Expanded*EntriesForTool` functions. baseDir is the homonto.toml directory
  (threaded in; the config already resolves relative content dirs against it).
- Expansion: process `local:` frameworks alongside `builtin:` — the expanded
  resource's `Source` is `builtin:<name>` in both cases (both materialize into
  the catalog root), so the projection path is unchanged.

## Engine

`materializeCatalog` (engine.go:232): build the catalog via the config's
`frameworkCatalog(...)` (base + locals) instead of `catalog.New()`, so a local
framework's skills materialize (from their srcFS) into the catalog root exactly
like builtins. Everything downstream (adapter linking `builtin:<name>`) is
unchanged.

## Acceptance test (the gate)

A `homonto.toml` with `[frameworks.myfw] source="local:./myfw"`, a `./myfw/`
dir (`framework.toml` name=myfw + `skills/myskill/SKILL.md`), driven through
`engine.Build`→`Plan`→`Apply`: `.homonto/catalog/skills/myskill/SKILL.md` exists
and the adapter links it. Plus: a non-`local:`/`builtin:` framework source still
fails at load; the whole builtin path is unchanged.

## Risk

Medium — new cross-subsystem behavior. Mitigations: the FS-aware index is
backward-identical for base-only catalogs (full suite green); the acceptance test
drives the real end-to-end path; de-globalizing `loadedCatalog` threads baseDir
explicitly (no hidden singleton state).

## Alternatives
- Require local framework dirs to be catalog-structured (`frameworks/<name>/`) —
  rejected; a standalone framework dir with framework.toml at its root is the
  natural, ergonomic layout.

```

## openspec/changes/local-frameworks/tasks.md

- Source: openspec/changes/local-frameworks/tasks.md
- Lines: 1-20
- SHA256: 092ab0667242e37e3c0100207850df9e00ae7f892080db5f729fd6301504d5ea

```md
# Tasks — local-frameworks

## 1. Catalog: FS-aware index + single-framework merge
- [ ] Track per-resource source FS (skillFS/commandFS/subagentFS); Materialize*
      + SubagentContent resolve from it (base-only identical). Add
      mergeFrameworkRoot + LoadWithLocal(base, locals). Tests: base identity;
      a local single-framework merges + materializes from its FS.

## 2. Config: local: acceptance + overlay catalog + expansion
- [ ] Accept local:<path> frameworks (keep F35 for other non-builtin); build the
      catalog with the config's local overlays (thread baseDir, replace the
      loadedCatalog singleton); expand local frameworks as builtin:<name>.

## 3. Engine wiring + E2E
- [ ] materializeCatalog builds the catalog with the config's local overlays.
      E2E: a local: framework's skill is materialized by apply; builtin path
      unchanged.

## 4. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/local-frameworks/specs/framework-expansion/spec.md

- Source: openspec/changes/local-frameworks/specs/framework-expansion/spec.md
- Lines: 1-28
- SHA256: d243c03bed636666cc8703040fc8fd12fb8598ff380a66b09e2515fda72219eb

```md
# framework-expansion

## ADDED Requirements

### Requirement: A local framework installs through the same validated path as a builtin

Config loading SHALL accept a framework whose source is `local:<path>`, where
`<path>` is a framework root (a `framework.toml` whose name equals the framework
key, plus its resources at framework-root-relative paths) resolved relative to
the config file. A `local:` framework MUST be validated through the same catalog
checks as a builtin (manifest schema, name-equals-key, resource-path existence,
dependency ranges) and its transitively-expanded resources MUST install through
the same projection and materialization path as a builtin framework's. Any other
non-builtin framework source MUST still fail loudly at load. A configuration
using only builtin frameworks MUST behave identically to before.

#### Scenario: A local framework's resource is installed by apply

- **GIVEN** a config declaring `[frameworks.myfw] source = "local:./myfw"` and a
  `./myfw` framework root providing a skill
- **WHEN** the change is applied
- **THEN** the skill is materialized and projected exactly as a builtin
  framework's skill would be

#### Scenario: A non-local, non-builtin framework source still fails

- **WHEN** a framework declares a source that is neither `builtin:` nor `local:`
- **THEN** loading fails loudly, unchanged from before

```
