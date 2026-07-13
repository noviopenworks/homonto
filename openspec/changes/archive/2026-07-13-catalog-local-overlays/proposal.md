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
