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
