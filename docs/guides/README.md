# Guides

User-facing documentation, one topic per file: `docs/guides/<topic>.md`.
Guides explain how to *use* the system; specs define what it *must do*.

## Obligation

Every onto change carries a `guides` obligation in its `state.yaml`
(`pending | updated | "waived: <reason>"` — the waiver is a quoted scalar).
`onto-close` refuses to archive a change while `guides: pending` — either
write/update the affected guide(s) or record an explicit waiver reason.

## Current Gap

The repo has an onto workflow guide but does not yet have a core homonto usage
guide. Future docs work should add a guide covering config schema, import
limitations, target names, pruning/state behavior, JSONC comment loss, and safe
secret handling.
