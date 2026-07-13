# Design — [compat].homonto

## Catalog (version-agnostic)

`frameworkTOML`: add `Compat struct { Homonto string } \`toml:"compat"\``.
`Framework`: add `Compat string` (= ft.Compat.Homonto). The catalog stores it but
does NOT evaluate it (it has no running-version knowledge; `internal/catalog`
must not import `internal/cli`). Add a loose comparator `satisfiesLoose(v, c)` =
`satisfies` after stripping any `-prerelease`/`+build` suffix from v, so
`0.1.0-dev` satisfies `>=0.1.0`.

## Engine (has the version)

`engine.Build` gains a trailing `homontoVersion string` parameter. After building
the framework catalog, for each `[frameworks.X]` the config declares, look up its
catalog `Framework`; if `Compat != ""`, require `satisfiesLoose(homontoVersion,
Compat)` — else return a clear "framework X requires homonto <constraint>, but
this is <version>" error (fail-closed, before any projection). Empty version
(unstamped/test default) skips the check.

`cli` passes `cli.Version` to `engine.Build` at its four call sites; tests pass a
version (buildEngine gets a fixed test version like "0.1.0").

## Consumer / test

A local framework declaring `[compat].homonto = ">=99.0.0"` fails to load under a
`0.1.0` homonto; `">=0.1.0"` loads. (`satisfiesLoose` unit-tested for the
pre-release strip.)

## Risk

Low logic; the `engine.Build` signature ripple (cli 4 sites + test helpers) is
mechanical and compiler-checked.

## Alternatives
- A leaf `internal/buildinfo.Version` imported by both cli and catalog — rejected
  here to avoid changing the release ldflags `-X` target (unverifiable in this
  env); the engine already sits above cli's version and below the catalog.
