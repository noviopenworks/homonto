# Design — framework capabilities

## Parsing

`frameworkTOML`: add `Provides struct { Capabilities []string } \`toml:"provides"\``
and `Capabilities []string \`toml:"capabilities"\`` inside the existing
`Dependencies` struct. `Framework` gains `Provides []string` and
`RequiredCapabilities []string`.

A capability string is `name@major`: split on the last `@`; `name` non-empty;
`major` a non-negative integer (reuse the int parse). A malformed capability is a
load error naming the framework.

## Resolution (after all sources merged)

In `validateDependencyRanges`'s sibling pass (or a new `validateCapabilities`
run right after it, once every framework — base + overlays — is indexed):
- Build `provided := set of every "name@major"` across all frameworks' Provides.
- For each framework, for each required capability, error if it is not in
  `provided`, naming the framework and the capability. Exact `name@major` match.
- Multiple providers are fine (a capability is an interface). Providing and
  requiring the same capability in one framework is allowed.

## Consumer

`catalog/frameworks/openspec/framework.toml`:
`[provides]\ncapabilities = ["spec-workflow@1"]`.
`catalog/frameworks/comet/framework.toml`:
`[dependencies]\ncapabilities = ["spec-workflow@1"]` (alongside its framework
deps). The embedded catalog must still load (openspec provides what comet
requires) — a test lowers/removes the provider to prove fail-loud.

## Risk

Low — additive; frameworks without capabilities are unchanged. The catalog +
expand suites and new capability tests pin it.

## Alternatives
- Major *ranges* (>=) for capabilities — deferred; exact `name@major` is the
  simplest useful contract (a major bump is an incompatible interface change).
