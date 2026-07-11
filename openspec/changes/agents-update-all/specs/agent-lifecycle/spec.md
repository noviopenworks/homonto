## ADDED Requirements

### Requirement: homonto agents update --all reconciles every installed agent

`homonto agents update --all` SHALL run the three-way merge (the same as `homonto
agents update <name>`) across every agent recorded in `.homonto/agents-lock.json`,
and print a summary of the outcome (merged/updated, up-to-date, conflicted,
skipped). An agent still declared in the config SHALL be merged exactly as the
single-agent update would; an installed agent no longer declared in the config
SHALL be skipped with a note; a per-agent error (e.g. a missing local source)
SHALL be reported for that agent without aborting the rest. The command SHALL exit
non-zero if any agent had a conflict or a per-agent error, and exit 0 when all
agents are clean. `agents update` with neither a name nor `--all`, or with both,
SHALL be a usage error; single `agents update <name>` behavior is unchanged.

#### Scenario: update --all merges every installed agent

- **GIVEN** two installed agents, one with a disjoint local+source edit and one already up-to-date
- **WHEN** `homonto agents update --all` runs
- **THEN** the first is auto-merged and the second reported up-to-date, a summary is printed, and the command exits 0

#### Scenario: update --all exits non-zero on any conflict

- **GIVEN** two installed agents, one of which has an overlapping (conflicting) edit
- **WHEN** `homonto agents update --all` runs
- **THEN** the conflicting agent gets a `.merged` sidecar and the command exits non-zero, while the other agent is still processed

#### Scenario: update --all skips an orphaned agent

- **GIVEN** an installed agent that is no longer declared in the config
- **WHEN** `homonto agents update --all` runs
- **THEN** it is skipped with a note and does not cause the whole run to fail (absent other issues, exit 0)

#### Scenario: name and --all are mutually exclusive

- **WHEN** `homonto agents update <name> --all` runs (or `agents update` with neither)
- **THEN** it is a usage error
