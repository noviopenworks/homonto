# Delta Spec: tool-adapters

## MODIFIED Requirements

### Requirement: Owned content linked by symlink with conflict detection

Owned skills SHALL be linked (not copied) from `content/skills/<name>` into
each tool's skills directory, and pending link work SHALL be visible as plan
changes: a missing link appears as a create, a link pointing at the wrong
target appears as an update, and a correct link is a no-op. `apply` SHALL
create the links even when they are the only pending changes. If the target
already exists and is not homonto's link, the adapter SHALL report a
conflict and SHALL NOT clobber it.

#### Scenario: Idempotent link creation

- **WHEN** a skill symlink does not yet exist
- **THEN** plan lists a create for that link, apply creates it, and a second
  plan/apply reports no change for that link

#### Scenario: Skills-only config still applies

- **GIVEN** a config whose only content is `[skills] own`
- **WHEN** the user runs `homonto apply` and confirms
- **THEN** the plan shows one create per missing link and apply creates
  every link (it does not short-circuit as "no changes")

#### Scenario: Relative content dir yields absolute link targets

- **GIVEN** homonto invoked from any working directory with a relative
  content dir (the default `content`)
- **WHEN** apply creates skill links
- **THEN** every symlink target is an absolute path resolved against the
  config file's directory, and the link does not dangle

#### Scenario: Conflict is reported, not clobbered

- **WHEN** the link target exists as a real file or points elsewhere
- **THEN** apply reports a conflict and leaves the existing file untouched
