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
