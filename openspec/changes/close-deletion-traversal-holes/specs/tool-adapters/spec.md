# tool-adapters (delta)

## ADDED Requirements

### Requirement: copy-mode prune is confined to the managed provider root

Copy-mode pruning SHALL delete only a destination that resolves under a managed
provider root. The adapter SHALL supply the set of managed roots (its user and,
when known, project provider directories) and the pruning step SHALL refuse to
delete — treating it as a prune failure that retains ownership — any recorded
destination that resolves outside every managed root (including via `..`
traversal). With an empty managed-root set the pruning step SHALL refuse every
delete (fail-closed). A tampered state entry whose recorded path points outside
the managed roots SHALL NOT cause an arbitrary file deletion.

#### Scenario: prune refuses an out-of-root destination

- **GIVEN** a recorded copy-mode ownership entry whose destination resolves outside the managed provider root (e.g. a tampered `state.json` path) but whose recorded hash matches the file
- **WHEN** apply prunes the de-declared resource
- **THEN** the out-of-root file is NOT deleted; the entry is reported as a prune failure and ownership is retained

#### Scenario: prune removes an in-root managed file

- **GIVEN** a de-declared copy-mode resource whose destination resolves under the managed provider root
- **WHEN** apply prunes it
- **THEN** the managed file is removed and its ownership record dropped
