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
