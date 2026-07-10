## ADDED Requirements

### Requirement: Bundled catalog embedded in binary

Homonto SHALL bundle a catalog directory tree at `catalog/` embedded in the Go binary via `go:embed`. The catalog SHALL contain framework metadata under `catalog/frameworks/<name>/framework.toml` and skill content under `catalog/skills/<name>/`. The embedded catalog SHALL be read-only at runtime.

#### Scenario: Catalog is available without external files

- **GIVEN** a Homonto binary built from a repo containing `catalog/`
- **WHEN** the binary runs on a machine without the source repo
- **THEN** the catalog frameworks and skills are accessible from the embedded filesystem

### Requirement: Builtin skill source resolution

A skill resource with `source = "builtin:<name>"` SHALL resolve its content from the embedded catalog at `catalog/skills/<name>/`. The content SHALL be materialized to `.homonto/catalog/skills/<name>/` on apply so that filesystem symlinks can point at a real directory.

#### Scenario: Builtin skill materializes on first apply

- **GIVEN** a config declaring `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** the user runs `homonto apply`
- **THEN** `.homonto/catalog/skills/brainstorming/` is created with content from the embedded catalog, and the skill is symlinked from there into the tool skills directories

#### Scenario: Builtin skill is idempotent on re-apply

- **GIVEN** a builtin skill already materialized and symlinked
- **WHEN** the user runs `homonto apply` again without any config change
- **THEN** no re-materialization occurs and the skill symlink is a no-op

### Requirement: Catalog version tracking

Homonto SHALL track the catalog version in `.homonto/state.json`. When the embedded catalog version differs from the recorded version, builtin resources SHALL be re-materialized on the next apply.

#### Scenario: Catalog upgrade triggers re-materialization

- **GIVEN** a builtin skill materialized under catalog version `0.1.0`
- **WHEN** a newer binary with catalog version `0.2.0` runs `homonto apply`
- **THEN** the builtin skill content in `.homonto/catalog/skills/<name>/` is refreshed and the state version is updated

### Requirement: Materialized catalog is generated state

The `.homonto/catalog/` directory SHALL be treated as generated cache. It SHALL NOT be committed to version control. The scaffolded `.gitignore` SHALL exclude `.homonto/` including the catalog cache.

#### Scenario: Gitignore covers catalog cache

- **GIVEN** a repo initialized with `homonto init`
- **WHEN** builtin skills are materialized to `.homonto/catalog/`
- **THEN** `git status` reports no untracked files under `.homonto/catalog/`

## MODIFIED Requirements

### Requirement: Local provider content root

Local provider content SHALL live under `homonto/` relative to the directory containing `homonto.toml`; generated state, cache, and the materialized builtin catalog SHALL live under `.homonto/` only. Current adapters resolve local-source skills (`source = "local:<name>"`) from `homonto/skills/<name>`. Builtin-source skills (`source = "builtin:<name>"`) resolve from the materialized `.homonto/catalog/skills/<name>/`. Local command, subagent, and framework content resolution is part of future framework/catalog projection work and MUST NOT be claimed as installed behavior yet.

#### Scenario: Local skill resolves from homonto/

- **GIVEN** a config with `[skills.my-skill] source = "local:my-skill"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `homonto/skills/my-skill/`

#### Scenario: Builtin skill resolves from materialized catalog

- **GIVEN** a config with `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `.homonto/catalog/skills/brainstorming/`
