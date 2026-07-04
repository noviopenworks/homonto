# Architecture Decision Records

`docs/adr/` holds **accepted or superseded** decisions only, one file per
decision: `NNNN-<slug>.md`.

## Staging rule

ADRs are drafted inside a change workspace
(`docs/changes/<name>/adr/<slug>.md`) with `Status: Proposed` and **no
number**. At close, `onto-close` assigns the next free global number and
moves the draft here with `Status: Accepted`. This keeps `docs/adr/` free of
abandoned-change noise and avoids number collisions between parallel changes.

## Numbering

- Four digits, zero-padded, strictly increasing: `0001`, `0002`, …
- The next number = highest existing number in `docs/adr/` + 1, assigned
  **only at close** by `onto-close`. Drafts in change workspaces are
  unnumbered (`<slug>.md`).
- Numbers are never reused. A superseded ADR keeps its file; its Status
  becomes `Superseded by NNNN`.

## Template

```markdown
# <Title, imperative: "Adopt X", "Use Y for Z">

- **Status:** Proposed | Accepted | Superseded by NNNN
- **Date:** YYYY-MM-DD
- **Change:** <change name that produced this decision>

## Context

What forces are at play; why a decision is needed.

## Decision

What we decided, stated actively ("We will …").

## Consequences

What becomes easier/harder; trade-offs accepted; follow-ups.
```
