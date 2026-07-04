# Architecture Decision Records

`docs/adr/` holds **accepted or superseded** decisions only, one file per
decision: `NNNN-<slug>.md`.

## Staging rule

ADRs are drafted inside a change workspace
(`docs/changes/<name>/adr/<slug>.md`) with `Status: Proposed` and **no
number**. At close, `onto-close` assigns the next free global number and
moves the draft here with `Status: Accepted`. This keeps `docs/adr/` free of
abandoned-change noise and avoids number collisions between parallel changes.
