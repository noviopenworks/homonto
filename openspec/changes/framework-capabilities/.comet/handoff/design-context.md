# Comet Design Handoff

- Change: framework-capabilities
- Phase: design
- Mode: compact
- Context hash: 2d7268e9ba6838c1b4056217602ef9f8a56fcdeef6d575b8255e97d06ed29bba

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/framework-capabilities/proposal.md

- Source: openspec/changes/framework-capabilities/proposal.md
- Lines: 1-41
- SHA256: 3b0341eeb9cb0251538a07b1a8a0ad6da70ed347b4307a37181ddc6635183581

```md
# Framework capabilities: depend on an interface, not a framework name

## Why

Roadmap E1 (F36), the capability model (design decision D2: `name@major`
capability strings). Now that frameworks can be shared (local + remote), a
framework should be able to depend on a *capability* (e.g. "spec-workflow@1")
that any provider satisfies, rather than hard-coding a specific framework name —
looser coupling for the ecosystem. A framework declares the capabilities it
provides and the capabilities it requires; the catalog resolves required
capabilities to providers at load, failing loud on an unresolved requirement.

## What Changes

- A `framework.toml` MAY declare `[provides].capabilities = ["name@major", …]`
  and `[dependencies].capabilities = ["name@major", …]`. A capability is a
  `name@major` string (name plus a non-negative integer major version).
- `catalog.Load`/`LoadOverlays`/`LoadWithLocal` validate the capability format
  and, after indexing all frameworks (base + overlays), resolve every required
  capability against the set provided across all frameworks — an unresolved
  requirement fails loud, naming the framework and the capability. Multiple
  providers of one capability are allowed (it is an interface, not a resource).
- Consumer: `openspec` provides `spec-workflow@1`; `comet` requires
  `spec-workflow@1` (it already depends on openspec by name — capabilities make
  the relationship interface-based).

## Impact

- **Specs:** `framework-expansion` gains a requirement that capability
  requirements are resolved fail-loud at load.
- **Behavior:** frameworks without capability declarations are unchanged; the
  new behavior is that a required capability with no provider fails at load.
- **Risk:** low — additive parsing + a load-time resolution pass; the catalog
  suite pins existing behavior.

## Non-goals

- Selecting *which* provider satisfies a capability when several do (any
  provider suffices; resource-name conflicts remain the strict-error case).
- `[compat].homonto` (needs version injection). Capability major-range
  requirements beyond exact `name@major` match.

```

## openspec/changes/framework-capabilities/design.md

- Source: openspec/changes/framework-capabilities/design.md
- Lines: 1-40
- SHA256: c0af9466681f305fc4bd9e55ce4ba2b3a43e1ddc514a291d56e680caa524fd5c

```md
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

```

## openspec/changes/framework-capabilities/tasks.md

- Source: openspec/changes/framework-capabilities/tasks.md
- Lines: 1-14
- SHA256: 5293249b864e4b372a5deba19a96b3df2e94e0559dc6aec66eb97a096ca0d6f5

```md
# Tasks — framework-capabilities

## 1. Capability parse + resolution
- [ ] frameworkTOML/[Framework] gain provides/required capabilities; parse+
      validate name@major; resolve required capabilities against the provided set
      across all merged sources fail-loud. TDD: unresolved capability errors;
      satisfied/absent load; cross-source (overlay) capability resolves.

## 2. Real consumer
- [ ] openspec provides spec-workflow@1; comet requires it. Embedded catalog
      loads; a test proves an unresolved requirement fails loud.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/framework-capabilities/specs/framework-expansion/spec.md

- Source: openspec/changes/framework-capabilities/specs/framework-expansion/spec.md
- Lines: 1-26
- SHA256: 78feea9829c1985553ced270d29659a07deef02e8d37e8faafc7c384702da063

```md
# framework-expansion

## ADDED Requirements

### Requirement: Framework capability requirements are resolved fail-loud

Catalog loading SHALL resolve every framework capability requirement against the
capabilities provided across all frameworks, where a capability is a `name@major`
string (a name plus a non-negative integer major). A framework MAY declare the
capabilities it provides and the capabilities it requires; a required capability
with no provider (exact `name@major` match) among the loaded frameworks MUST fail
the load with an error naming the framework and the capability. A malformed
capability string MUST fail loud. Multiple providers of one capability are
permitted (a capability is an interface, not a uniquely-owned resource), and a
framework with no capability declarations MUST behave exactly as before.

#### Scenario: An unresolved capability requirement fails to load

- **WHEN** a framework requires a capability that no loaded framework provides
- **THEN** loading fails with an error naming the framework and the capability

#### Scenario: A satisfied capability requirement loads

- **WHEN** a required capability is provided by some loaded framework (including
  one merged from an overlay)
- **THEN** the catalog loads and expansion behaves as before

```
