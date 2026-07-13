# onto-binary

## ADDED Requirements

### Requirement: onto abandon marks a change as an unsuccessful terminal

`onto abandon <change>` SHALL mark a change abandoned — the unsuccessful terminal
state, distinct from the successful `close`/archived terminal. It MUST enforce the
same preconditions as other lifecycle commands (framework gate, valid change name,
loadable `onto-state.yaml`), then set the change's `abandoned` flag and write
nothing else. It MUST be idempotent (abandoning an already-abandoned change
succeeds and changes nothing) and MUST refuse to abandon a change that is already
`archived` (a completed change is not abandonable), writing nothing.

Once a change is abandoned it is terminal: `onto advance` MUST refuse to advance an
abandoned change, leaving its phase unchanged. `onto graph` MUST surface the
abandoned state — `abandoned: true` in `--json` and an `abandoned` marker in the
human listing — so a cancelled change is never presented as ordinary active work.

#### Scenario: abandon marks the change

- **WHEN** `onto abandon <change>` runs on a loadable change that is not archived
- **THEN** the change's `onto-state.yaml` records `abandoned: true`, leaving its
  phase and other fields unchanged

#### Scenario: advance refuses an abandoned change

- **GIVEN** an abandoned change
- **WHEN** `onto advance` runs on it
- **THEN** it refuses naming the change as abandoned and leaves the phase unchanged

#### Scenario: abandon refuses an archived change

- **GIVEN** a change already `archived`
- **WHEN** `onto abandon` runs on it
- **THEN** it refuses and writes nothing
