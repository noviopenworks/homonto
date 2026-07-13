# Comet Design Handoff

- Change: framework-dependency-ranges
- Phase: design
- Mode: compact
- Context hash: 78af88f114a4776aee52d81eb768ed56f706c3ca8d90ca7805be482087245791

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/framework-dependency-ranges/proposal.md

- Source: openspec/changes/framework-dependency-ranges/proposal.md
- Lines: 1-46
- SHA256: a7d75beeb162a0ed9c46ee345fa5e576cdb5683fc8ea8517c9f59457aa4094ea

```md
# Framework dependency version ranges, validated fail-loud

## Why

Roadmap E1 (F36), phase-2 of the ecosystem-model design. A framework's
`[dependencies].frameworks` are today bare names with no version constraint, so
a framework that requires a particular version of a dependency cannot say so and
an incompatible pairing would only surface (if ever) at runtime. The E1 exit
gate wants compatibility to "fail loudly." This change adds **dependency version
ranges** validated at catalog load — the first compatibility mechanism, with a
real consumer (the built-in `comet` framework depends on `superpowers`).

Decision taken (per the E1 design's recommended defaults): use a **minimal
hand-rolled `x.y.z` comparator** rather than adding a semver dependency — the
project deliberately minimizes its module graph (pinned toolchains for
govulncheck) and framework versions are simple three-part versions.

## What Changes

- A dependency entry MAY be `"name@<constraint>"` where `<constraint>` is a
  comparator over `x.y.z` (`>=`, `>`, `<=`, `<`, `=`, or a bare exact version).
  A bare `"name"` (today's form) means any version — fully backward-compatible.
- `catalog.Load` validates, after indexing all frameworks, that every ranged
  dependency's target exists and its `version` satisfies the constraint; an
  unsatisfied or unparseable constraint fails loudly, naming the framework, the
  dependency, and the versions.
- The dependency graph used for expansion continues to key on the **name**
  (constraint stripped), so cycle detection and transitive resolution are
  unchanged.
- `comet`'s manifest declares `superpowers@>=0.1.0` and `openspec@>=0.1.0` as the
  first real consumer.

## Impact

- **Specs:** `framework-expansion` gains a requirement that dependency version
  ranges are validated fail-loud at load.
- **Behavior:** bare-name deps behave exactly as today; the only new behavior is
  that a ranged dependency whose target version is out of range fails at load.
- **Risk:** low — additive parsing + a small pure comparator (thoroughly tested)
  + one load-time check; the catalog suite is the safety net.

## Non-goals

- `[compat].homonto` ranges, capabilities, local/custom resolution (later E1
  phases, some gated on D1/D2).
- Pre-release / build-metadata semver (framework versions are plain `x.y.z`).

```

## openspec/changes/framework-dependency-ranges/design.md

- Source: openspec/changes/framework-dependency-ranges/design.md
- Lines: 1-52
- SHA256: 0e9491aff7b4f887a20f335e0daae88739e3c89762f9cfd2915eb07caf4eaddb

```md
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

```

## openspec/changes/framework-dependency-ranges/tasks.md

- Source: openspec/changes/framework-dependency-ranges/tasks.md
- Lines: 1-14
- SHA256: a6d230667c594d4e9ff9a1718efded01ea466c0862e0f17a557c0d58dccfa09e

```md
# Tasks — framework-dependency-ranges

## 1. Comparator + dep-range validation
- [ ] Add a pure x.y.z comparator (parseVer/satisfies) + unit tests. Parse
      "name@constraint" deps (bare name = any), carry constraints, and validate
      at catalog.Load fail-loud (unknown dep / out-of-range / unparseable).
      Cycle/transitive expansion unchanged (keys on name).

## 2. Real consumer
- [ ] comet manifest declares superpowers@>=0.1.0, openspec@>=0.1.0. Catalog
      loads (both at 0.1.0). Tests prove out-of-range fails loud.

## 3. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/framework-dependency-ranges/specs/framework-expansion/spec.md

- Source: openspec/changes/framework-dependency-ranges/specs/framework-expansion/spec.md
- Lines: 1-28
- SHA256: e3d3035cde8ad26622de0290cb48bccc814e1d336bea6b22d2eb54bb7974698b

```md
# framework-expansion

## ADDED Requirements

### Requirement: Framework dependency version ranges are validated fail-loud

Catalog loading SHALL validate every constrained framework dependency, where a
dependency of the form `"name@<constraint>"` compares the target framework's
three-part `x.y.z` version against `<constraint>` (`>=`, `>`, `<=`, `<`, `=`, or
a bare exact version). The dependency framework MUST exist and its version MUST
satisfy the constraint, otherwise loading fails with an error naming the
framework, the dependency, the version, and the constraint. A bare dependency
name (no constraint) MUST mean any version, preserving existing behavior, and the
dependency graph used for transitive resolution and cycle detection MUST continue
to key on the dependency name.

#### Scenario: An out-of-range dependency fails to load

- **WHEN** a framework declares a dependency `"dep@>=2.0.0"` and the indexed
  `dep` framework is version `1.0.0`
- **THEN** catalog loading fails with an error naming the framework, `dep`, the
  version, and the constraint

#### Scenario: A satisfied or bare dependency loads

- **WHEN** a dependency constraint is satisfied by the target's version, or the
  dependency is a bare name with no constraint
- **THEN** the catalog loads and transitive resolution behaves as before

```
