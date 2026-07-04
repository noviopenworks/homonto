# Symlink owned content; merge unowned keys surgically; never clobber

- **Status:** Accepted
- **Date:** 2026-07-03
- **Change:** homonto-v1-core

## Context

Users author skills/commands/rules in the repo (`content/`) but tools load
them from their own directories; copying would create stale duplicates.
Tool config files also contain keys homonto does not manage, which must
survive every apply.

## Decision

We will symlink owned content from `content/<kind>/<name>` into each tool's
directory, and never clobber: a destination that exists and is not our
symlink is reported as a conflict, not overwritten. For JSON/JSONC config we
write only managed keys and preserve every unmanaged key on merge.

## Consequences

- Editing `content/...` is instantly live in every tool.
- Conflicts surface loudly and are resolved by humans.
- JSONC inline comments inside rewritten regions may not survive — a known,
  documented limitation.
- Symlink targets must be absolute to be valid from anywhere (enforced
  since add-onto-workflow).
