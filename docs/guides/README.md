# Guides

User-facing documentation, one topic per file: `docs/guides/<topic>.md`.
Guides explain how to *use* the system; specs define what it *must do*.

## Obligation

Every onto change carries a `guides` obligation in its `state.yaml`
(`pending | updated | waived: <reason>`). `onto-close` refuses to archive a
change while `guides: pending` — either write/update the affected guide(s)
or record an explicit waiver reason.
