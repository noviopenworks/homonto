# framework-expansion

## ADDED Requirements

### Requirement: The catalog can merge validated overlay framework sources

Catalog construction SHALL support merging one or more overlay framework sources
over a base source, validating every source through the same checks (manifest
schema, name-equals-directory, resource-path existence, dependency ranges). An
overlay that redefines a resource name already provided by an earlier source with
a different path MUST be rejected (strict conflict policy); an identical
name-to-path mapping collapses idempotently. Loading with no overlays MUST be
identical to loading the base source alone, and dependency-range validation MUST
run once after all sources are indexed so a cross-source dependency is checked.

#### Scenario: An overlay adds a framework over the base

- **WHEN** the catalog is loaded with a base source and an overlay providing a
  new framework
- **THEN** both frameworks and their resources are indexed and expandable

#### Scenario: An overlay may not shadow a base resource

- **WHEN** an overlay declares a resource name already provided by the base with
  a different path
- **THEN** loading fails with a conflict error

#### Scenario: No overlays is identical to loading the base alone

- **WHEN** the catalog is loaded with no overlays
- **THEN** the result is identical to loading the base source by itself
