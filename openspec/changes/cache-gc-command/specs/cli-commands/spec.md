# cli-commands (delta)

## ADDED Requirements

### Requirement: cache gc reclaims unreferenced remote cache entries

`homonto cache gc [--dry-run]` SHALL reclaim content-addressed remote cache entries
that no entry in the remote lockfile references, and SHALL report the digests it
removed. With `--dry-run` it SHALL report what it would remove without deleting
anything. The command SHALL reject stray positional arguments.

#### Scenario: dry-run reports without deleting

- **WHEN** the user runs `homonto cache gc --dry-run`
- **THEN** it reports the unreferenced entries it would reclaim and deletes nothing
