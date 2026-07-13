# apply-pipeline (delta)

## ADDED Requirements

### Requirement: control-plane files are written no-follow under the .homonto root

`homonto` SHALL write its own control-plane files (state, cache, lockfile, and
catalog materialization under `.homonto/`) without following a symlink at the
destination. The writer SHALL refuse to write when the target path is, or resolves
through, a symlink, and SHALL confine the resolved path under the project's
`.homonto` root;
a target escaping that root SHALL be a write error, never followed. Writes to a
tool's own config files (which may legitimately be user-symlinked) keep the
existing atomic writer.

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
