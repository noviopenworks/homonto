---
comet_change: catalog-foundation-skills
role: technical-design
canonical_spec: openspec
---

# Catalog Foundation (Skills) — Technical Design

Deep technical refinement of the open-phase `design.md` (D1–D5). OpenSpec delta
specs (`builtin-catalog`, `framework-expansion`, `config-model`, `tool-adapters`)
remain the canonical WHAT; this document is the HOW, grounded in the current Go
code (`internal/{config,engine,adapter,link,state,skillpath}`). It does not
change scope; it fixes the one non-compiling detail in the open phase (task 2.1)
and specifies exact package boundaries, signatures, data flow, and tests.

## 1. Package architecture

The frozen `config-model`/`builtin-catalog` specs SHALL-mandate the catalog at
`catalog/` at the repo root, embedded via `go:embed`. Go's `//go:embed` cannot
reach a parent/sibling directory, so the embed directive lives in a package
rooted at `catalog/`. Task 2.1's plan (`//go:embed all:catalog` inside
`internal/catalog/catalog.go`) cannot compile and is corrected here.

```
catalog/                      package catalog  (embedded content + FS only)
  embed.go                    //go:embed all:frameworks all:skills → var FS embed.FS
  version.txt                 single catalog version string, e.g. "0.1.0"
  frameworks/<name>/framework.toml
  skills/<name>/SKILL.md, references/…

internal/catalog/             package catalog  (logic; imports .../catalog for FS)
  catalog.go     Load(fs), Framework/Skill types, index, Version()
  expand.go      dependency graph, cycle detection, transitive expansion, dedup
  materialize.go Materialize(dstRoot, names) — embedded FS → .homonto/catalog/skills/

internal/config/              imports internal/catalog (one-way)
internal/engine/              imports config + adapters + catalog materialization
internal/adapter/{claude,opencode}   gain a catalogRoot
internal/link/                managed-root check generalized to a set
internal/state/               gains CatalogVersion
```

**Layering rule (prevents an import cycle):** `internal/catalog` MUST NOT import
`internal/config`. It exposes config-agnostic types (framework name, version,
dependency names, skill name→catalog-path map). `internal/config` imports
`internal/catalog` and builds `config.NamedResource` values itself. The root
`catalog` package imports nothing but `embed`.

## 2. Root `catalog` package

`catalog/embed.go`:

```go
package catalog

import "embed"

//go:embed all:frameworks all:skills version.txt
var FS embed.FS
```

`all:` includes dotfiles and `_`-prefixed files so skill `references/` are not
silently dropped. The package exports only `FS`; all logic lives in
`internal/catalog`. It is importable (not under `internal/`) but exports only an
embedded filesystem, which is an acceptable, inert surface.

## 3. `internal/catalog`

### Types

```go
type Framework struct {
    Name         string
    Version      string
    Description  string
    Dependencies []string          // framework names
    Skills       map[string]string // skill name → catalog-relative path ("skills/<n>")
}

type Catalog struct {
    fs         fs.FS
    frameworks map[string]Framework
    version    string
}
```

### Loading (`catalog.go`)

`Load(fsys fs.FS) (*Catalog, error)` walks `frameworks/*/framework.toml`,
unmarshals each via `github.com/pelletier/go-toml/v2` (already a dependency),
indexes by `Name`, and reads `version.txt` (trimmed). A production
constructor `New()` calls `Load(catalog.FS)`; tests pass an `fstest.MapFS`.
Validation at load: every `framework.toml` `[skills]` path SHALL exist in the
embedded FS (guards against a metadata/content drift — task 1.4); a framework's
declared `name` SHALL equal its directory name.

`SkillContentFS(name string) (fs.FS, error)` returns a sub-FS rooted at the
skill's catalog path for the materializer to copy.

### Expansion + cycle detection (`expand.go`)

```go
// Expand returns the transitive, deduplicated set of skill names reachable
// from the given framework names, or an error naming a dependency cycle.
func (c *Catalog) Expand(frameworkNames []string) (skills []ExpandedSkill, err error)

type ExpandedSkill struct {
    Name      string // skill resource name
    Framework string // origin framework (for plan-origin notes later)
}
```

DFS over the framework dependency graph with a three-color (white/grey/black)
visitor: encountering a grey node is a cycle → error listing the chain
(`A → B → A`). Skills accumulate into an ordered set keyed by skill name;
duplicates (a skill reachable via two frameworks) collapse to one entry. Output
is sorted by skill name for deterministic plans. Expansion is pure graph logic
with no filesystem or config dependency, so it unit-tests against a small
in-memory `Catalog`.

### Materialization (`materialize.go`)

```go
// Materialize extracts each named builtin skill from the embedded FS into
// dstRoot/<name>/, replacing any existing content for that skill. It is the
// caller's job (engine) to gate this on version.
func (c *Catalog) Materialize(dstRoot string, skillNames []string) error
```

Per skill: resolve its catalog path, `fs.WalkDir` the sub-FS, recreate
directories (0755) and write files (0644) under `dstRoot/<name>/`. To make an
interrupted run detectable (Spec Patch #2), the engine records the catalog
version in state ONLY after `Materialize` returns nil for all skills; a crash
mid-extract leaves state's version unchanged, so the next apply re-materializes.
Within a skill, extraction removes the skill's existing dir first
(`os.RemoveAll(dstRoot/<name>)`) then re-writes, so a stale file from a previous
version cannot survive an upgrade.

## 4. `internal/config` integration

`config` imports `internal/catalog`. `config.Load` stays a pure parse +
validate; framework expansion is exposed through a new method so adapters get
effective skills without the engine re-plumbing:

```go
func (c *Config) ExpandedSkillEntriesForTool(tool string) ([]NamedResource, error)
```

Algorithm:
1. Start from explicit `SkillEntriesForTool(tool)` (existing behavior).
2. For each `[frameworks.<fw>]` whose `source` is `builtin:<fw>` and whose
   `TargetsOrAll()` contains `tool`, call `catalog.Expand([fw])`.
3. Each expanded skill becomes a `NamedResource{Name: skill, Resource:{Source:
   "builtin:"+skill, Scope: fwResource.Scope, Targets: fwResource.Targets}}`
   — inheriting the framework declaration's scope and targets (Spec Patch #1).
4. **Collision:** an expanded skill name equal to an explicit `[skills.X]` name,
   or to a skill from a different framework with a conflicting declaration, is an
   error naming the collision (framework-expansion "Framework atomicity").
5. **Cycle:** surfaced from `catalog.Expand` as a `Load`-class error.

A package-level singleton `catalog.New()` is built once (embedded FS is cheap to
index); `ExpandedSkillEntriesForTool` uses it. Errors from expansion are returned
(not panicked) so `plan`/`apply` report them cleanly.

Adapters switch their one call site from `c.SkillEntriesForTool("claude")` to
`c.ExpandedSkillEntriesForTool("claude")` and propagate the error out of `Plan`.

## 5. `internal/link` — managed roots as a set

Today `managed(target, contentRoot)` checks a single prefix. Builtin skill links
target `.homonto/catalog/skills/…`, a second managed root. Generalize:

```go
func managed(target string, roots ...string) bool // true if under ANY root
func Plan(srcs map[string]string, roots ...string) ([]Op, error)
func Link(src, dst string, roots ...string) (bool, error)
func Remove(dst string, roots ...string) error
func IsManaged(dst string, roots ...string) bool
```

Variadic keeps every existing single-root call site source-compatible
(`Plan(m, a.content)` still compiles). Adapters pass both roots
(`a.content, a.catalogRoot`). This is the minimal change that lets conflict
detection, relink, and prune treat a link into either managed tree as "ours"
while still refusing to touch a user's foreign file or link.

## 6. `internal/state` — catalog version slot

The catalog version is global, not per-tool, so it does not fit the `Managed`
map. Add a top-level field:

```go
type State struct {
    Managed        map[string]map[string]Entry `json:"managed"`
    CatalogVersion string                      `json:"catalogVersion,omitempty"`
}
```

`omitempty` keeps existing `state.json` files backward-compatible (absent =
empty = "force materialize"). Two small accessors: `CatalogVersionRecorded()
string` and `SetCatalogVersion(v string)`.

## 7. `internal/engine` — materialization orchestration

`Build` computes `catalogRoot := filepath.Join(stateDir, "catalog", "skills")`
(`<configdir>/.homonto/catalog/skills`) and threads it into both adapters via a
new `WithCatalogRoot(catalogRoot)` (mirrors `WithProjectRoot`).

Materialization runs in `Apply`, before the adapter loop, so no symlink is
created ahead of its target:

```go
func (e *Engine) Apply(sets []adapter.ChangeSet) error {
    // ... existing secret pre-resolve ...
    if err := e.materializeCatalog(); err != nil { return err }
    // ... existing adapter apply loop ...
}
```

`materializeCatalog`:
1. Collect the set of builtin skill names the config actually declares
   (union across tools of `ExpandedSkillEntriesForTool` entries whose source is
   `builtin:`).
2. If none → return (no catalog work).
3. If `state.CatalogVersion == catalog.Version()` AND every skill dir exists →
   skip (version-gated).
4. Otherwise `catalog.Materialize(catalogRoot, names)`, then
   `state.SetCatalogVersion(catalog.Version())`. State is saved by the existing
   `Apply` tail / per-adapter saves; add an explicit save after materialization
   so a later adapter failure still records the completed materialization.

`Plan` does not materialize (it must not write). `link.Plan` only stats the link
destinations, not sources, so a plan computed before materialization is correct;
apply materializes before linking.

## 8. Adapters (`claude`, `opencode`)

Add a `catalogRoot` field and `WithCatalogRoot`. `links()` and the adopt/prune
paths resolve source by scheme:

```go
func (a *Adapter) skillSource(entry config.NamedResource) string {
    if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
        return filepath.Join(a.catalogRoot, strings.TrimPrefix(s, "builtin:"))
    }
    return filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, entry.Name))
}
```

Every `link.*` call passes both roots: `link.Plan(a.links(), a.content,
a.catalogRoot)`, `link.Link(src, dst, a.content, a.catalogRoot)`,
`link.IsManaged(p, a.content, a.catalogRoot)`, `link.Remove(dst, a.content,
a.catalogRoot)`. The existing local behavior is unchanged; builtin links become
first-class. `ObserveHashes` is unaffected (it reads the recorded `dst -> src`
link regardless of which managed tree `src` is in). Both adapters share the same
change (opencode mirrors claude). Identical changes go to `util.go` helpers if
the source join is duplicated there.

`doctor` (engine `status.go`): for a recorded builtin `skill.<n>`, verify the
materialized target exists under `catalogRoot`; a missing materialized dir is
reported like a broken link, prompting a re-apply.

## 9. Data flow (apply, `[frameworks.comet]`)

```
config.Load(homonto.toml)
  → ExpandedSkillEntriesForTool: [frameworks.comet] → catalog.Expand(comet)
      → comet's 8 skills + superpowers + openspec skills, deduped, scope/targets
        inherited from [frameworks.comet]
engine.Apply
  → materializeCatalog: version-gated extract of those skills to
    .homonto/catalog/skills/<n>/  → record CatalogVersion
  → claude.Apply / opencode.Apply: create symlinks dst → .homonto/catalog/skills/<n>,
    record skill.<n> in state
status/doctor: drift = link target changed OR materialized dir missing
```

## 10. Error handling & edge cases

- **Metadata/content drift** — `catalog.Load` fails fast if a `framework.toml`
  skill path is absent from the embedded FS (catches a bad catalog at build/test).
- **Cycle** — reported by name from `Expand`, surfaced through `config`.
- **Collision** — expanded-vs-explicit name clash is an error before any write.
- **Interrupted materialization** — version recorded only after full success →
  next apply re-materializes (Spec Patch #2).
- **Version upgrade** — per-skill `RemoveAll` before re-extract prevents stale
  files surviving an upgrade.
- **Conflict** — a real file or foreign symlink at a builtin skill's link dst is
  reported, never clobbered (unchanged `link` semantics, now with two roots).
- **Gitignore** — `.homonto/` is already ignored; confirm the scaffolded
  `.gitignore` covers `.homonto/catalog/` (config-model "generated state").

## 11. Testing strategy

- `internal/catalog`: `fstest.MapFS` fixtures for parse, missing-skill-path
  failure, transitive expansion, dedup, three cycle shapes, and version read;
  `Materialize` into `t.TempDir()` incl. upgrade-removes-stale and nested
  `references/`.
- `internal/config`: `ExpandedSkillEntriesForTool` expansion, scope/target
  inheritance, collision error, cycle error — driven from the real embedded
  catalog.
- `internal/link`: multi-root `managed`/`Plan`/`Link`/`Remove`/`IsManaged`
  (target under root A, under root B, under neither).
- adapters: builtin link create, idempotent re-apply, prune of de-declared
  builtin skill (managed accepts catalogRoot), conflict-not-clobbered, state
  `skill.<n>` recorded — for both claude and opencode.
- `internal/engine`: first-apply materialization, version-gated skip,
  version-change refresh, partial-materialization re-run.
- Regression + dogfood: `go test ./... -count=1`, `go vet`, `go build`, then
  `homonto apply --yes` / `status` / `doctor` on a `[frameworks.comet]` config
  (task group 6).

## 12. Spec Patches applied

1. `framework-expansion` — expanded skills inherit the framework declaration's
   `scope`/`targets` (requirement text + new scenario).
2. `builtin-catalog` — catalog version recorded only after materialization
   completes; partial extraction re-materializes (requirement text + new
   scenario).

Both are supplementary acceptance scenarios / boundary conditions; no scope
change.
