# Guides

User-facing documentation, one topic per file: `docs/guides/<topic>.md`.
Guides explain how to *use* the system; specs define what it *must do*.

## Obligation

Every onto change carries a `guides` obligation in its `state.yaml`
(`pending | updated | "waived: <reason>"` — the waiver is a quoted scalar).
`onto-close` refuses to archive a change while `guides: pending` — either
write/update the affected guide(s) or record an explicit waiver reason.

## Current Guides

- [`using-homonto.md`](using-homonto.md) covers the core CLI, config shape,
  projection behavior, status/adoption, and known limitations.
- [`status-and-adoption.md`](status-and-adoption.md) explains state adoption,
  drift, pending changes, and pruning behavior.
- [`comet-workflow.md`](comet-workflow.md) documents this repository's current
  development workflow (Comet + OpenSpec + Superpowers).
- [`onto-workflow.md`](onto-workflow.md) is the legacy Onto workflow guide,
  retained for historical context and potential product-framework reference.

Release-readiness tasks live in [`../road-to-release.md`](../road-to-release.md).
