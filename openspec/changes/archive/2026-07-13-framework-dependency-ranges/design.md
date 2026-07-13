# Design — framework dependency version ranges

## Comparator (minimal, hand-rolled, pure)

New `internal/catalog/version.go` (or inline):
```go
// parseVer parses "x.y.z" into [3]int; error on malformed.
func parseVer(s string) ([3]int, error)
// satisfies reports whether version v meets constraint c, where c is one of
// ">=x.y.z", ">x.y.z", "<=x.y.z", "<x.y.z", "=x.y.z", or a bare "x.y.z" (exact).
func satisfies(v, c string) (bool, error)
```
Comparison is lexicographic over the three ints. Only these operators; anything
else is a parse error (fail loud, not silently pass). Pure and fully unit-tested
(equal, gt/lt each component, boundary, malformed).

## Dependency parsing

`[dependencies].frameworks` entries split on the last `@`:
- `"superpowers"` → name=superpowers, constraint="" (any).
- `"superpowers@>=0.1.0"` → name=superpowers, constraint=">=0.1.0".

`Framework.Dependencies` keeps the **names** (constraint stripped) so
`expandResources`' cycle/transitive walk is unchanged. A parallel
`Framework.DependencyConstraints map[string]string` (name→constraint) carries the
ranges for validation.

## Load-time validation

After all frameworks are indexed in `catalog.Load`, a final pass: for each
framework fw, for each (depName, constraint) with a non-empty constraint:
- if depName is not an indexed framework → error (unknown dependency);
- parse the dep's `version` and the constraint; if unparseable → error;
- if not `satisfies` → error naming fw, depName, the dep version, and the
  constraint.

Bare-name deps skip the version check (any version) — today's behavior.

## Consumer

`catalog/frameworks/comet/framework.toml`:
`frameworks = ["superpowers@>=0.1.0", "openspec@>=0.1.0"]`. Both are at 0.1.0,
so the check passes; a test lowers/raises to prove fail-loud both ways.

## Risk

Low — additive; bare deps unchanged; the comparator is small and pure. The
catalog + expand suites and new comparator tests pin it.

## Alternatives
- Add golang.org/x/mod/semver — rejected here to keep the module graph minimal
  (project pins toolchains for govulncheck); plain x.y.z needs no library.
