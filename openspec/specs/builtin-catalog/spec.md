# builtin-catalog Specification

## Purpose
TBD - created by archiving change catalog-foundation-skills. Update Purpose after archive.
## Requirements
### Requirement: Catalog loading from embedded filesystem

Homonto SHALL load the bundled catalog from the embedded Go filesystem at startup. The catalog loader SHALL parse all `catalog/frameworks/<name>/framework.toml` files and index frameworks by name. The loader SHALL provide framework lookup, skill content path resolution, and dependency graph traversal.

#### Scenario: Load all frameworks

- **GIVEN** a binary with four framework.toml files in the embedded catalog
- **WHEN** the catalog is loaded
- **THEN** four frameworks are indexed by name and each has its metadata parsed

### Requirement: Skill content materialization

Homonto SHALL materialize builtin skill content from the embedded catalog to `.homonto/catalog/skills/<name>/` before creating symlinks. Materialization SHALL be version-aware: the catalog version is tracked in state, and re-materialization occurs only when the version changes or the directory is missing. The catalog version SHALL be recorded in state only after materialization completes, so an interrupted or partial extraction is never mistaken for an up-to-date cache.

#### Scenario: First materialization

- **GIVEN** no `.homonto/catalog/` directory exists
- **WHEN** apply runs with a config declaring builtin skills
- **THEN** all declared builtin skills are extracted to `.homonto/catalog/skills/<name>/`

#### Scenario: Version-gated re-materialization

- **GIVEN** `.homonto/catalog/` exists with state recording catalog version `0.1.0`
- **WHEN** apply runs with the same binary (same version)
- **THEN** materialization is skipped and existing directories are reused

#### Scenario: Partial materialization is not recorded as current

- **GIVEN** a prior apply whose materialization was interrupted before the catalog version was recorded in state
- **WHEN** the next apply runs
- **THEN** the catalog is re-materialized (state's recorded version does not match the embedded version) rather than trusting the incomplete cache
