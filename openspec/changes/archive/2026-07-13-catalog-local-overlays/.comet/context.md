# Comet Design Handoff

- Change: catalog-local-overlays
- Phase: design
- Mode: compact
- Context hash: 1c4d5063e77ed0d869914b931ebeb7cb674b2f23fdab86e6f61403004cce4235

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/catalog-local-overlays/proposal.md

- Source: openspec/changes/catalog-local-overlays/proposal.md
- Lines: 1-45
- SHA256: 9941fe9f136aed3f67d6ecffea870f4cb162178c6d2d4ceac5e4a39f9e2670ab

```md
# Catalog overlay loading — the foundation for local/custom frameworks

## Why

Roadmap E1 (F36), the local/custom-framework mechanism (design decision D1:
`local:<path>` frameworks are structurally validated — the user already controls
their own filesystem, so a local framework is no less trusted than their
`homonto.toml`; no digest required, mirroring `local:` skill/content sources).

The catalog is loaded once from the embedded FS (`catalog.New()` /
`config.loadedCatalog()`), so a framework declared on the local filesystem has
nowhere to be read from. Supporting local frameworks end-to-end needs, first, a
**catalog that can merge additional (local) framework sources over the embedded
base** with the same validation and an explicit conflict policy. This change
delivers that foundational mechanism; the config `local:` acceptance and engine
materialization that consume it follow in the next phased changes.

## What Changes

- Add `catalog.LoadOverlays(base fs.FS, overlays ...fs.FS) (*Catalog, error)` —
  loads the base catalog, then merges each overlay filesystem's frameworks and
  resources through the **same** validation `Load` already applies (name==dir,
  paths exist, manifest schema, dependency ranges).
- **Strict conflict policy** (design decision D3): an overlay that redefines a
  resource name already provided by the base (or an earlier overlay) with a
  different path is a hard error — an overlay may not silently shadow a builtin.
  (Identical name→path is the existing idempotent collapse.)
- `catalog.Load(fsys)` becomes `LoadOverlays(fsys)` with no overlays — behavior
  unchanged.

## Impact

- **Specs:** `framework-expansion` gains a requirement that the catalog can merge
  validated overlay framework sources under a strict conflict policy.
- **Behavior:** none today — `New()`/`Load` are unchanged; the new entry point
  has no caller until the config/engine wiring lands. It is the tested
  building block, exactly as `structproj` shipped before its adapters consumed it.
- **Risk:** low — additive; the existing catalog suite pins base behavior, new
  tests pin the overlay merge + conflict policy.

## Non-goals

- Config `local:<path>` framework acceptance and engine materialization of local
  resources (the next phased changes that consume this).
- Remote/digest-pinned frameworks (a later phase via the trust pipeline).

```

## openspec/changes/catalog-local-overlays/design.md

- Source: openspec/changes/catalog-local-overlays/design.md
- Lines: 1-60
- SHA256: 663fa2be4ca8eaf92596029a67a5a2fa1375bcd714510e9c31913493d62f42d9

```md
# Design — catalog overlay loading

## Refactor Load into a merge over one-or-more sources

Today `Load(fsys)` indexes one FS. Generalize to merge several, base first:

```go
func Load(fsys fs.FS) (*Catalog, error) { return LoadOverlays(fsys) }

func LoadOverlays(base fs.FS, overlays ...fs.FS) (*Catalog, error) {
    c := newCatalog()
    for _, src := range append([]fs.FS{base}, overlays...) {
        if err := c.mergeSource(src); err != nil { return nil, err }
    }
    // dependency-range validation runs once, after ALL sources are indexed
    return c, c.validateDependencyRanges()
}
```

`mergeSource(src)` is today's per-source body (version.txt only from base;
loose skills/commands/subagents; per-framework manifest parse + manifest-schema
guard + name==dir + path existence + resource indexing). The existing
"resource name mapped to two different paths → error" check already implements
the **strict conflict policy** across sources — an overlay redefining a builtin
skill name to a different path hits exactly that error. Same name→same path
collapses (idempotent).

`version.txt` is read only from the base (overlays are frameworks, not a whole
catalog release); if an overlay lacks it, that's fine — only the base's is the
catalog version.

## Conflict policy (D3 strict) — already enforced

The `if prev, ok := c.skills[name]; ok && prev != sp` guard (and the command/
subagent equivalents), applied while merging each source in order, IS the strict
policy: a builtin resource cannot be shadowed by an overlay with different
content. No new code — it falls out of merging into the shared index.

## Dependency-range validation timing

Must run AFTER all sources are indexed (an overlay framework may depend on a base
framework, or vice-versa), so it moves from inside the per-source loop to a final
pass over `c.frameworks` (it already is a post-loop pass; keep it after the merge
loop).

## Test

- `LoadOverlays(base)` == `Load(base)` (no overlay → unchanged; the whole
  existing suite still passes).
- base + an overlay MapFS adding a new framework → both frameworks present, the
  overlay's skills indexed and expandable.
- an overlay redefining a base skill name to a different path → strict-conflict
  error.
- an overlay framework depending on a base framework with a satisfied range →
  loads; unsatisfied → fails (validation sees both sources).

## Risk

Low — a pure structural refactor of `Load` into `mergeSource` + a variadic entry
point; base behavior identical (pinned by the existing suite).

```

## openspec/changes/catalog-local-overlays/tasks.md

- Source: openspec/changes/catalog-local-overlays/tasks.md
- Lines: 1-12
- SHA256: 8a6dbacb9ae29d6d3c74a5cb5993c6529201d081ec221680e2bed56e85adf1bf

```md
# Tasks — catalog-local-overlays

## 1. Overlay merge
- [ ] Refactor Load into mergeSource + LoadOverlays(base, overlays...); Load(fsys)
      = LoadOverlays(fsys) (behavior unchanged). Dependency-range validation runs
      once after all sources merge. Strict conflict falls out of the shared-index
      guard. TDD: no-overlay identity; overlay adds a framework; overlay shadow
      conflict errors; cross-source dependency range validated.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      existing catalog suite unchanged.

```

## openspec/changes/catalog-local-overlays/specs/framework-expansion/spec.md

- Source: openspec/changes/catalog-local-overlays/specs/framework-expansion/spec.md
- Lines: 1-31
- SHA256: be2626e004507662cfb845bfef28e90e99822e8edc13af0c8245158eff831065

```md
# framework-expansion

## ADDED Requirements

### Requirement: The catalog can merge validated overlay framework sources

Catalog construction SHALL support merging one or more overlay framework sources
over a base source, validating every source through the same checks (manifest
schema, name-equals-directory, resource-path existence, dependency ranges). An
overlay that redefines a resource name already provided by an earlier source with
a different path MUST be rejected (strict conflict policy); an identical
name-to-path mapping collapses idempotently. Loading with no overlays MUST be
identical to loading the base source alone, and dependency-range validation MUST
run once after all sources are indexed so a cross-source dependency is checked.

#### Scenario: An overlay adds a framework over the base

- **WHEN** the catalog is loaded with a base source and an overlay providing a
  new framework
- **THEN** both frameworks and their resources are indexed and expandable

#### Scenario: An overlay may not shadow a base resource

- **WHEN** an overlay declares a resource name already provided by the base with
  a different path
- **THEN** loading fails with a conflict error

#### Scenario: No overlays is identical to loading the base alone

- **WHEN** the catalog is loaded with no overlays
- **THEN** the result is identical to loading the base source by itself

```
