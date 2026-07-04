# Write files atomically and persist state last

- **Status:** Accepted
- **Date:** 2026-07-03
- **Change:** homonto-v1-core

## Context

An apply interrupted mid-write (crash, ctrl-C, power loss) must never leave
a tool config half-written or homonto's state claiming work that did not
happen.

## Decision

We will write every file atomically (temp file + rename in the same
directory) and write `.homonto/state.json` last, after all tool files
succeeded. State therefore only ever describes writes that really landed.

## Consequences

- An interrupted apply leaves each tool file either old or new — never torn.
- Worst case after interruption: state is stale-old, so the next plan shows
  already-applied changes again; re-apply is idempotent and converges.
- Every writer must go through the shared atomic-write helper; direct
  `os.WriteFile` on managed files is a defect.
