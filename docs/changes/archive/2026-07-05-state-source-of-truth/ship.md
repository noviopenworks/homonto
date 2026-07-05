## Make state.json the source of truth for adoption and drift

homonto's `state.json` already stored an `Applied` hash of each managed key's
last-applied value, but the non-secret code path ignored it — so imported or
pre-existing resources looked managed yet escaped pruning and drift, and
`homonto status` reported un-applied config edits as if the disk had drifted.

### What changed

- New silent `adopt` action: a declared, non-secret key already matching disk
  (or recorded with a stale hash) is recorded/refreshed in state on apply — no
  tool-file write, no diff line, no prompt; reported as "Reconciled N…".
- `apply` writes a tool file only when a managed key in it changed, so
  adoption leaves `.claude.json`/`settings.json`/`opencode.jsonc` byte-identical
  (comments preserved).
- `homonto status` now compares disk against the last-applied `Applied` hash and
  reports un-applied config edits separately as "N config change(s) awaiting
  apply" instead of as drift.

### Verification

Full mode, Result: pass. 10 delta-spec scenarios evidenced; two parallel
adversarial skeptics refuted/broke nothing. Regression: 125 tests + race + vet +
gofmt clean.

Full records: `docs/changes/archive/2026-07-05-state-source-of-truth/`
(proposal · design · verification · notes)

### Branch

`feature/20260705/state-source-of-truth` (base `a72e535`), not yet merged to
`main`.
