## MODIFIED Requirements

### Requirement: Owned content linked by symlink with conflict detection

Owned skills SHALL be linked (not copied) from their source into each tool's skills directory at the location chosen by the skill resource's `scope`, and pending link work SHALL be visible as plan changes: a missing link appears as a create, a link pointing at the wrong target appears as an update, and a correct link is a no-op. Local-source skills (`source = "local:<name>"`) SHALL be linked from `homonto/skills/<name>`. Builtin-source skills (`source = "builtin:<name>"`) SHALL be linked from the materialized catalog at `.homonto/catalog/skills/<name>`. `apply` SHALL create both local and builtin skill links even when they are the only pending changes, and SHALL record each applied link in state (`skill.<name>`: desired target path plus applied hash) so drift detection and pruning both see it. A skill removed from the config SHALL have its link pruned only when the existing path is a symlink pointing into homonto's managed content (either `homonto/skills/` for local or `.homonto/catalog/skills/` for builtin). If the target already exists and is not homonto's link, the adapter SHALL report a conflict and SHALL NOT clobber it -- for creation and for pruning alike.

#### Scenario: Idempotent link creation

- **WHEN** a skill symlink does not yet exist
- **THEN** plan lists a create for that link, apply creates it, and a second plan/apply reports no change for that link

#### Scenario: Skills-only config still applies

- **GIVEN** a config whose only content is owned skills declared as explicit `[skills.<name>]` resources
- **WHEN** the user runs `homonto apply` and confirms
- **THEN** the plan shows one create per missing link and apply creates every link

#### Scenario: Relative local content dir yields absolute link targets

- **GIVEN** homonto invoked from any working directory with the default `homonto/` local provider root
- **WHEN** apply creates skill links
- **THEN** every symlink target is an absolute path resolved against the config file's directory, and the link does not dangle

#### Scenario: Builtin skill links to materialized catalog

- **GIVEN** a config with `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is the absolute path to `.homonto/catalog/skills/brainstorming/`

#### Scenario: Conflict is reported, not clobbered

- **WHEN** the link target exists as a real file or points elsewhere
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: Applied link recorded in state

- **WHEN** apply creates a skill link
- **THEN** state contains a `skill.<name>` record, and `homonto status` reports drift if the link is later changed out-of-band

#### Scenario: De-declared skill pruned only when it is our link

- **GIVEN** a skill resource removed from `homonto.toml` whose target path is a real file (or a symlink pointing outside homonto's managed roots)
- **WHEN** apply processes the resulting delete
- **THEN** the path is left untouched and a conflict is reported; only a symlink into homonto's managed content (`homonto/skills/` or `.homonto/catalog/skills/`) is removed
