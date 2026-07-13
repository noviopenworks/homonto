## Why

Two engine security holes (ROADMAP N4, gate B; T-hostile — the projector consumes
untrusted config/state and deletes files):

- **F7 (arbitrary deletion):** copy-mode prune removes `op.Dst` taken from the
  recorded `Entry.Desired` path in `homonto/state.json` (`internal/copyfile/
  copyfile.go:145`). A tampered state entry whose recorded hash matches lets prune
  delete an arbitrary file. Destinations must be reconstructed from validated
  resource identity and confined under the managed provider root.
- **F28 (local: traversal):** `validateResources` (skills/commands,
  `internal/config/config.go:693`) does NOT apply the `local:` plain-name check
  that `validateSubagents` already applies (`config.go:740`). A
  `local:../../x` skill/command source passes validation and joins a traversal
  suffix into a provider path. Skills and commands must get the same plain-name
  validation subagents have.

## What Changes

- `validateResources` rejects a `local:` source that is not a plain name (no
  `.`/`..`/path separators), mirroring `validateSubagents`.
- Copy-mode prune confines its delete target under the managed provider root: a
  reconstructed/validated destination outside the root is refused, never deleted.

## Impact

- **Code:** `internal/config/config.go` (validateResources), `internal/copyfile/`
  and/or the copy-mode adapters (prune destination confinement) + tests.
- **Spec:** `config-model` (local: plain-name for skills/commands) and
  `tool-adapters` (prune root confinement) deltas.
- **Out of scope:** N5/N6 (remote transactionality, no-follow writer, locking).
