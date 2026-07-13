## Why

ROADMAP E2 / finding F48 (safe core): `homonto import --force` overwrites an
existing `homonto.toml` in place (`os.WriteFile`, `internal/cli/import.go:31`).
If the import is degraded (a source tool config was skipped/warned), `--force`
can replace a valid, hand-tuned config with a worse one, unrecoverably. Import
should back up the existing config before overwriting and write atomically.

(The stronger F48 change — making a source PARSE failure fatal before any
destination mutation — conflicts with the existing cli-commands requirement that
an unreadable tool file *warns* rather than omits; that semantic change is a
separate design decision and is out of scope here.)

## What Changes

- Before `import --force` overwrites an existing config, copy it to
  `<config>.bak`; write the new config atomically (`fsutil.WriteAtomic`).

## Impact

- **Code:** `internal/cli/import.go` + test.
- **Spec:** `cli-commands` delta (import backs up before overwrite).
- **Out of scope:** source-parse-fatal semantics (spec conflict, deferred).
