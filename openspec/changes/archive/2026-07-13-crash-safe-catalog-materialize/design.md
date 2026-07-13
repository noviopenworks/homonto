# Design — crash-safe catalog materialization

## Approach

In `catalog.Catalog.Materialize`, per skill:
1. `staging := dstDir + ".staging"`; `os.RemoveAll(staging)` first (discard any
   leftover from a prior crash).
2. Walk the embedded skill FS writing into `staging` (same `WriteControlPlane`
   no-follow writes as today, just rooted at `staging`).
3. On walk success: `os.RemoveAll(dstDir)` then `os.Rename(staging, dstDir)`.
4. On any walk error: return it; `staging` is left for the next run's step-1
   cleanup, and `dstDir` is untouched (still the prior complete version).

Commands/subagents are unchanged (already atomic single-file writes).

## Why this is sufficient

- Mid-walk failure → `dstDir` intact (old complete version), no partial dst.
- Crash in the RemoveAll→Rename window → `dstDir` absent (not partial), so
  `allSkillDirsExist` returns false and re-materializes next run.
- `Rename` within the same `.homonto` parent is atomic on POSIX.
- No change to the success-path bytes, so all existing materialize tests pass.

## Risk / safety

Localized to one function. The staging dir lives under the same control-plane
root as the destination, so `WriteControlPlane`'s no-follow guarantee still
holds. New test drives a mid-walk failure (an unreadable/oversized entry or an
injected error) and asserts the destination is never partial.

## Alternatives

- Completion marker / content-hash in `allSkillDirsExist` — rejected; atomic
  swap makes directory presence a sufficient completeness signal without a
  second gate to keep in sync.
