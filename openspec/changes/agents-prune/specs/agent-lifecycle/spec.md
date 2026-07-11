## ADDED Requirements

### Requirement: homonto agents prune removes stale managed installs

`homonto agents prune` SHALL remove homonto-managed agent installs that are no
longer declared and drop their lockfile records. It SHALL handle:

- an **orphan agent** (recorded in `.homonto/agents-lock.json` but not declared in
  the config): remove each recorded target install file and drop the agent's
  lockfile entry;
- a **de-declared target** (a target in an agent's `Installed` no longer in the
  agent's declared targets): remove that target's install file and drop its
  `Installed` entry, keeping the agent and its still-declared targets.

It SHALL only touch a file at a homonto-recorded install path. Before removing a
file whose on-disk content differs from the recorded base hash (a local edit), it
SHALL back the file up to `<path>.bak`. It SHALL also remove a pruned target's
leftover `<path>.merged` sidecar. It SHALL report each pruned item and print
`nothing to prune` when there is nothing to remove. A `--dry-run` flag SHALL list
what would be pruned and change nothing.

#### Scenario: prune an orphaned agent

- **GIVEN** an agent recorded in the lockfile that is no longer declared in the config
- **WHEN** `homonto agents prune` runs
- **THEN** its recorded install files are removed and its lockfile entry is dropped

#### Scenario: prune a de-declared target

- **GIVEN** an agent whose lockfile records a target the agent no longer declares
- **WHEN** `homonto agents prune` runs
- **THEN** that target's install file is removed and its `Installed` entry dropped, while the agent and its still-declared targets remain

#### Scenario: prune backs up a locally-modified install

- **GIVEN** an orphan agent whose install file was locally edited (differs from the recorded base hash)
- **WHEN** `homonto agents prune` runs
- **THEN** the file is copied to `<path>.bak` before being removed

#### Scenario: nothing to prune

- **GIVEN** a workspace whose lockfile matches the config exactly
- **WHEN** `homonto agents prune` runs
- **THEN** it reports nothing to prune and changes nothing

#### Scenario: dry run changes nothing

- **GIVEN** an orphan agent
- **WHEN** `homonto agents prune --dry-run` runs
- **THEN** it lists the orphan as prunable but removes no files and does not change the lockfile
