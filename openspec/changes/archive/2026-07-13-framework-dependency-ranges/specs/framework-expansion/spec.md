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
