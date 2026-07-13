# cli-commands (delta)

## ADDED Requirements

### Requirement: init augments an existing .gitignore

`homonto init` SHALL, when a `.gitignore` already exists in the target directory,
augment it with any missing homonto entries (`/.homonto/`, `.env`) while preserving
the existing content, rather than skipping it — so a repository that already has a
`.gitignore` still ignores homonto's control-plane state and secrets. It SHALL
report augmented files distinctly from newly created ones.

#### Scenario: an existing .gitignore is augmented, not skipped

- **GIVEN** a directory with a `.gitignore` containing `node_modules/`
- **WHEN** `homonto init` runs
- **THEN** the `.gitignore` still contains `node_modules/` and now also contains `/.homonto/` and `.env`, and init reports it as updated
