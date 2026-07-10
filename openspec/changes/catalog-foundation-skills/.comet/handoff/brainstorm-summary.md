# Brainstorm Summary

- Change: catalog-foundation-skills
- Date: 2026-07-10

## Confirmed Technical Approach

Deep design grounded in the current Go code (config, engine, adapters, link,
state, skillpath). Honors the frozen open-phase artifacts (proposal, design.md
D1–D5, delta specs); no scope rewrite.

**Package layout (CONFIRMED by user — "Root catalog pkg, honor spec"):**
- `catalog/` at repo root (per frozen config-model/builtin-catalog SHALL):
  - `catalog/embed.go` = `package catalog`, `//go:embed all:frameworks all:skills`,
    exports `var FS embed.FS`. (Corrects task 2.1, whose `//go:embed all:catalog`
    inside `internal/catalog/` cannot compile — go:embed can't reach a parent dir.)
  - `catalog/frameworks/<n>/framework.toml`, `catalog/skills/<n>/…`
  - `catalog/version.txt` — single catalog version string.
- `internal/catalog/` = logic package, imports `.../catalog` for `catalog.FS`:
  - `catalog.go` — load embedded FS, parse framework.toml, index frameworks,
    skill-path resolution, version read.
  - `expand.go` — dependency graph, cycle detection, transitive expansion, dedup.
  - `materialize.go` — extract builtin skill from embedded FS to
    `.homonto/catalog/skills/<n>/`, version-gated.
  - **Zero dependency on `internal/config`** (avoids an import cycle; returns
    config-agnostic types — framework/skill/dep names).

**Config integration (`internal/config`):**
- `config` imports `internal/catalog` (one-way; catalog never imports config).
- `Config.ExpandedSkillEntriesForTool(tool)` = explicit `[skills.X]` PLUS
  framework-expanded skills. Framework expansion: `[frameworks.X] source =
  "builtin:<fw>"` → catalog.Expand(fw) → transitive skill names → each becomes a
  builtin-source skill NamedResource that **inherits the framework declaration's
  scope and targets**.
- Collision (framework skill name == explicit `[skills.X]`) → `Load`/expansion error.
- Dependency cycle → error naming the chain (detected in `internal/catalog`).

**Adapters (`claude`, `opencode`):**
- Add a `catalogRoot` field (`.homonto/catalog/skills`) alongside `content`.
- Adapters call `ExpandedSkillEntriesForTool` (not `SkillEntriesForTool`).
- `links()` resolves builtin src → `catalogRoot/<name>`, local src → `content/skills/<name>`.

**link package:**
- Generalize the single `contentRoot` prefix check to a set of managed roots
  (`content` + `catalogRoot`); `managed()` is true if target is under ANY root.
  Ripples through `Plan`/`Link`/`Remove`/`IsManaged` signatures + call sites.

**state package:**
- Add a top-level `CatalogVersion string` field (backward-compatible, omitempty)
  as the global materialization-version slot (per-tool `managed` map is unsuitable).

**Engine:**
- `Engine.Apply` materializes all declared builtin skills BEFORE the adapter loop,
  gated on `state.CatalogVersion == embedded version`; on mismatch/first run,
  (re)extract then record the new version. `Build` threads `catalogRoot` into adapters.
- Materialization order matters: materialize first so adapter symlinks never dangle.

## Key Trade-offs and Risks

- Root `catalog/` is a Go package holding only an embedded FS — accepted to honor
  the frozen `catalog/`-at-root spec (the alternative, `internal/catalog/content/`,
  would need a spec re-open; user chose to honor the spec).
- link-package signature change touches both adapters + their tests — modest ripple.
- config→internal/catalog dependency is new; kept acyclic by making catalog
  config-agnostic.
- Materialization adds an apply step — idempotent, version-gated, runs only on
  first apply or version change.

## Testing Strategy

- `internal/catalog`: framework.toml parse; transitive expansion; dedup; cycle
  detection; materialization into `t.TempDir()`; version read/compare. Drive from
  the real embedded catalog + a small in-test `fstest.MapFS` for edge cases.
- `internal/config`: expansion via `ExpandedSkillEntriesForTool`, collision error,
  cycle error, scope/target inheritance.
- adapters: builtin link create (src = catalogRoot), idempotent re-apply, prune of
  de-declared builtin skill (managed accepts catalogRoot), conflict-not-clobbered,
  state `skill.<name>` recorded.
- engine: first-apply materialization, version-gated skip, version-change refresh.
- Full regression: `go test ./... -count=1`, `go vet`, `go build`, plus dogfood
  `homonto apply --yes` / `status` / `doctor` on the real `[frameworks.comet]` config.

## Spec Patches

Proposed (supplementary acceptance scenarios / boundary conditions only — no scope
change), pending user confirmation:

1. **framework-expansion** — add scenario: framework-expanded skills inherit the
   `[frameworks.X]` declaration's `scope` and `targets` (governs where each skill
   links and which tools receive it). Currently unspecified.
2. **builtin-catalog** — add boundary scenario: a partial/failed materialization
   (e.g. interrupted extract) is not recorded as the current catalog version, so
   the next apply re-materializes rather than treating an incomplete cache as done.

(If the user prefers to defer these, they can be dropped without affecting scope.)
