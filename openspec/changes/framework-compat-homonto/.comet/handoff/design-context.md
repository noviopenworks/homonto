# Comet Design Handoff

- Change: framework-compat-homonto
- Phase: design
- Mode: compact
- Context hash: 9745b625b60545d7dc19c2ae2fe759faf1dd9eabd1c8de836e81ecf5a7ea7f63

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/framework-compat-homonto/proposal.md

- Source: openspec/changes/framework-compat-homonto/proposal.md
- Lines: 1-36
- SHA256: 47c4915f80142df5d8d11a38e70c1dbda880c6ae6f9f769f26f17576fc215c53

```md
# Framework [compat].homonto — fail loud on an incompatible homonto version

## Why

Roadmap E1 (F36), the last compatibility mechanism. Now that frameworks can be
shared (local + remote), a framework built for a newer homonto could be
installed on an older one and misbehave silently. A framework should declare the
homonto version range it supports and fail loud at load if the running homonto
is outside it.

## What Changes

- A `framework.toml` MAY declare `[compat].homonto = "<constraint>"` (a version
  constraint over `x.y.z`, reusing the existing comparator; pre-release/build
  metadata on the running version is stripped for the comparison, so a
  `0.1.0-dev` build satisfies `>=0.1.0`).
- The catalog parses it into `Framework.Compat` (staying version-agnostic — it
  does not know the running version).
- The **engine** checks each declared framework's `Compat` against the running
  homonto version (injected into `engine.Build`) and fails closed with a clear
  error before any projection when the version is unsatisfied. A framework with
  no `[compat]` is unconstrained (unchanged).

## Impact

- **Specs:** `framework-expansion` gains a requirement that a framework's
  homonto-compat range is enforced fail-loud.
- **Behavior:** frameworks without `[compat]` are unchanged; new: an incompatible
  framework fails at load with guidance.
- **Risk:** low-per-change but the `engine.Build` version parameter ripples to
  its callers (cli + tests) mechanically.

## Non-goals

- Framework-to-framework compat beyond dependency version ranges (already
  shipped). F38.

```

## openspec/changes/framework-compat-homonto/design.md

- Source: openspec/changes/framework-compat-homonto/design.md
- Lines: 1-38
- SHA256: 19a9d1accb991f6efc3a29aa5bd27dc1fd9048bb8ef95239da6918430cd3838c

```md
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

```

## openspec/changes/framework-compat-homonto/tasks.md

- Source: openspec/changes/framework-compat-homonto/tasks.md
- Lines: 1-14
- SHA256: bd6fd1cb9f9a8e1cdeefe597d3da85fae0eed30fe1cb68166e95e6bfc753a763

```md
# Tasks — framework-compat-homonto

## 1. Catalog Compat field + loose comparator
- [ ] frameworkTOML/[Framework] gain Compat (from [compat].homonto); catalog
      stays version-agnostic. Add satisfiesLoose (strip pre-release/build). Unit
      tests for satisfiesLoose.

## 2. Engine version check + cli wiring
- [ ] engine.Build gains homontoVersion; checks each declared framework's Compat
      fail-closed. cli passes cli.Version (4 sites); test helpers pass a version.
      E2E: [compat].homonto=">=99.0.0" fails; ">=0.1.0" loads.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/framework-compat-homonto/specs/framework-expansion/spec.md

- Source: openspec/changes/framework-compat-homonto/specs/framework-expansion/spec.md
- Lines: 1-25
- SHA256: d14182ac5a40b42400302bc356d534ef8a9087e17cb190d5ea90fe8f5fa6fb98

```md
# framework-expansion

## ADDED Requirements

### Requirement: A framework's homonto compatibility range is enforced fail-loud

homonto SHALL enforce a framework's declared homonto compatibility range: a
`framework.toml` MAY declare `[compat].homonto` as a version constraint over
`x.y.z`, and when a declared framework's constraint is not satisfied by the
running homonto version, loading MUST fail closed with an error naming the
framework, its constraint, and the running version, before any projection. Any
pre-release or build-metadata suffix on the running version MUST be ignored for
the comparison (a development build of a version satisfies that version's
constraints). A framework with no `[compat]` declaration MUST be unconstrained,
unchanged from before.

#### Scenario: An incompatible framework fails to load

- **WHEN** a declared framework requires a homonto version the running binary does not satisfy
- **THEN** loading fails closed naming the framework, the constraint, and the running version

#### Scenario: A compatible or unconstrained framework loads

- **WHEN** a framework's homonto constraint is satisfied, or it declares no `[compat]`
- **THEN** it loads and installs as before

```
