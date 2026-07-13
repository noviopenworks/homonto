# apply-pipeline (delta)

## ADDED Requirements

### Requirement: control-plane files are written no-follow under the .homonto root

`homonto` SHALL write its own control-plane files (state, cache, lockfile, and
catalog materialization under `.homonto/`) without following a symlink at the
destination. The writer SHALL refuse to write when the destination's final path
component is a symlink (it never resolves it), so a symlink planted at a
predictable `.homonto` control-plane path cannot redirect the write outside the
project. Writes to a tool's own config files (which may legitimately be
user-symlinked) keep the existing atomic writer.

#### Scenario: a symlinked control-plane target is refused

- **GIVEN** a `.homonto` control-plane path whose final component is a symlink pointing outside `.homonto`
- **WHEN** `homonto` writes that control-plane file
- **THEN** the write is refused with a clear error and the symlink target is not modified

#### Scenario: a normal control-plane write succeeds

- **GIVEN** a `.homonto` control-plane path that is a regular file or absent
- **WHEN** `homonto` writes it
- **THEN** the write succeeds atomically under the `.homonto` root

### Requirement: apply takes a project-scoped exclusive lock

`homonto apply` SHALL acquire a project-scoped exclusive lock (under `.homonto`)
before mutating state or tool files, and SHALL release it on exit. A second
concurrent `apply` on the same project SHALL fail fast with a clear "another apply
is in progress" error rather than racing to a last-writer-wins outcome.

#### Scenario: a second concurrent apply fails fast

- **GIVEN** an `apply` holding the project lock
- **WHEN** a second `apply` starts on the same project
- **THEN** it exits non-zero reporting that another apply is in progress, and does not mutate state
