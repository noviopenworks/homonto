# Architecture Decision Records

`docs/adr/` holds **accepted or superseded** decisions only, one file per
decision: `NNNN-<slug>.md`. It is decision *history* — a record of what was
decided and why, kept even after a decision is superseded.

## Staging rule

New ADRs are drafted inside an active OpenSpec change (staged with the change's
design artifacts, `Status: Proposed`, and no number). The number is assigned
only when the change is archived, which keeps `docs/adr/` free of
abandoned-change noise and avoids collisions between parallel changes.

## Numbering

- Four digits, zero-padded, strictly increasing: `0001`, `0002`, …
- The next number = highest existing number in `docs/adr/` + 1, assigned when
  the producing change is archived.
- Numbers are never reused. A superseded ADR keeps its file; its Status becomes
  `Superseded by NNNN`.

## Template

```markdown
# <Title, imperative: "Adopt X", "Use Y for Z">

- **Status:** Proposed | Accepted | Superseded by NNNN
- **Date:** YYYY-MM-DD
- **Change:** <change name that produced this decision>

## Context / ## Decision ("We will …") / ## Consequences
```
